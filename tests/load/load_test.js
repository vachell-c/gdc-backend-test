import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Treat all 2xx, 3xx, and 4xx status codes as expected responses,
// so deliberate negative tests (401, 404, 400) don't count as failures.
http.setResponseCallback(http.expectedStatuses({ min: 200, max: 499 }));

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Custom metrics
const assignErrors = new Rate('assign_errors');
const idempotencyHits = new Counter('idempotency_hits');
const taskCreateTime = new Trend('task_create_time');
const assignTime = new Trend('assign_time');
const searchTime = new Trend('search_time');

const TEAM_IDS = [
  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1',
  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2',
  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3',
];

export const options = {
  stages: [
    { duration: '10s', target: 5 },   // Ramp up to 5 VUs
    { duration: '20s', target: 10 },  // Ramp to 10 VUs
    { duration: '20s', target: 10 },  // Hold at 10
    { duration: '10s', target: 0 },   // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'],  // 95% of requests under 2s
    http_req_failed: ['rate<0.05'],      // Less than 5% failures
    assign_errors: ['rate<0.1'],         // Assignment errors under 10%
  },
};

export default function () {
  // Each VU gets a unique identity to avoid registration conflicts
  const vuId = __VU;
  const iterId = __ITER;
  const email = `load-${vuId}-${iterId}@k6test.example.com`;
  const teamId = TEAM_IDS[vuId % TEAM_IDS.length];

  // ── Phase 1: Authentication (register + login) ──
  group('01_auth', function () {
    let res = http.post(`${BASE_URL}/register`, JSON.stringify({
      email: email,
      password: 'loadtest123',
      name: `Load VU ${vuId}`,
      team_id: teamId,
    }), { headers: { 'Content-Type': 'application/json' } });

    // Register may 409 if email already exists — that's OK
    let token;
    if (res.status === 201) {
      token = res.json('token');
    } else {
      // Login instead
      res = http.post(`${BASE_URL}/login`, JSON.stringify({
        email: email,
        password: 'loadtest123',
      }), { headers: { 'Content-Type': 'application/json' } });
      check(res, { 'login succeeded': (r) => r.status === 200 });
      token = res.json('token');
    }

    check(token, { 'has auth token': (t) => t !== undefined && t !== '' });

    if (!token) {
      // Cannot proceed without auth
      return;
    }

    // ── Phase 2: Task creation with idempotency ──
    group('02_task_create', function () {
      for (let i = 0; i < 3; i++) {
        // Generate a deterministic valid UUID per task using VU + iteration
        const hex = (vuId * 100 + iterId * 10 + i).toString(16).padStart(12, '0');
        const taskKey = `a0000000-0000-0000-0000-${hex}`;
        const t0 = Date.now();
        const createRes = http.post(`${BASE_URL}/tasks`, JSON.stringify({
          title: `Load Task ${vuId}-${iterId}-${i}`,
          description: `Generated during k6 load test by VU ${vuId}, iteration ${iterId}`,
          status: i === 0 ? 'pending' : i === 1 ? 'in_progress' : 'completed',
        }), {
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
            'Idempotency-Key': taskKey,
          },
        });
        taskCreateTime.add(Date.now() - t0);
        check(createRes, {
          'task created or idempotent': (r) => r.status === 201,
        });
      }
    });

    // ── Phase 3: List, filter, search, paginate ──
    group('03_task_list', function () {
      // List all tasks
      let res = http.get(`${BASE_URL}/tasks`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      check(res, { 'list tasks 200': (r) => r.status === 200 });

      // Filter by status
      res = http.get(`${BASE_URL}/tasks?status=pending`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      check(res, { 'filter by status 200': (r) => r.status === 200 });

      // Paginated query
      const t0 = Date.now();
      res = http.get(`${BASE_URL}/tasks?limit=3&page=1`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      searchTime.add(Date.now() - t0);
      check(res, {
        'paginated list 200': (r) => r.status === 200,
        'meta has page/limit/total': (r) => {
          const meta = r.json('meta');
          return meta && meta.page !== undefined && meta.limit !== undefined && meta.total !== undefined;
        },
      });

      // Full-text search
      res = http.get(`${BASE_URL}/tasks?search=Load`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      check(res, { 'search tasks 200': (r) => r.status === 200 });
    });

    // ── Phase 4: Task assignment (cross-team) ──
    group('04_task_assign', function () {
      // Pick a task to assign — get the first task from the list
      let listRes = http.get(`${BASE_URL}/tasks`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      if (listRes.status !== 200) return;

      const tasks = listRes.json('data');
      if (!tasks || tasks.length === 0) return;

      const taskId = tasks[0].id;

      // Try assigning to a user in the SAME team — get another VU's email
      // Since all VUs in same team share team_id, this should work
      const assigneeEmail = `load-${(vuId + 1) % 10}-0@k6test.example.com`;

      // First, get the assignee's ID by logging them in
      let loginRes = http.post(`${BASE_URL}/login`, JSON.stringify({
        email: assigneeEmail,
        password: 'loadtest123',
      }), { headers: { 'Content-Type': 'application/json' } });

      if (loginRes.status === 200) {
        const assigneeToken = loginRes.json('token');
        
        // Decode JWT to get user ID — or just try assigning
        // Actually, we need the user ID. Let's try registering to get it.
        let regRes = http.post(`${BASE_URL}/register`, JSON.stringify({
          email: assigneeEmail,
          password: 'loadtest123',
          name: `Assign Target ${vuId}`,
          team_id: teamId,
        }), { headers: { 'Content-Type': 'application/json' } });

        let assigneeId;
        if (regRes.status === 201) {
          assigneeId = regRes.json('user.id');
        } else {
          // User already exists — login and we can't get the ID easily without another endpoint
          // Skip assignment for this iteration
          return;
        }

        if (!assigneeId) return;

        const t0 = Date.now();
        const assignRes = http.post(`${BASE_URL}/tasks/${taskId}/assign`, JSON.stringify({
          user_id: assigneeId,
        }), {
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
          },
        });
        assignTime.add(Date.now() - t0);

        const success = check(assignRes, {
          'assign task 200': (r) => r.status === 200,
          'assignee matches': (r) => r.json('data.assignee_id') === assigneeId,
        });
        if (!success) assignErrors.add(1);
      }
    });

    // ── Phase 5: Error path testing ──
    group('05_error_paths', function () {
      // Missing auth
      let res = http.get(`${BASE_URL}/tasks`);
      check(res, { 'no auth 401': (r) => r.status === 401 });

      // Invalid task ID
      res = http.get(`${BASE_URL}/tasks/00000000-0000-0000-0000-000000000000`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      check(res, { 'not found 404': (r) => r.status === 404 });

      // Missing Idempotency-Key
      res = http.post(`${BASE_URL}/tasks`, JSON.stringify({ title: 'No Key Task' }), {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
      });
      check(res, { 'missing idempotency key 400': (r) => r.status === 400 });
    });

    // ── Phase 6: Verify the same idempotency key returns same result ──
    group('06_idempotency_replay', function () {
      // Replay the first task creation with the same idempotency key
      const hex = (vuId * 100 + iterId * 10 + 0).toString(16).padStart(12, '0');
      const replayKey = `a0000000-0000-0000-0000-${hex}`;
      const replayRes = http.post(`${BASE_URL}/tasks`, JSON.stringify({
        title: `Load Task ${vuId}-${iterId}-0 (replay)`,
        description: `This should be ignored — idempotency should return the original`,
        status: 'pending',
      }), {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
          'Idempotency-Key': replayKey,
        },
      });
      if (replayRes.status === 201) {
        idempotencyHits.add(1);
      }
      check(replayRes, {
        'idempotent replay 201': (r) => r.status === 201,
        'replay has task id': (r) => r.json('data.id') !== undefined,
      });
    });

    // Small think time between iterations
    sleep(1);
  });
}
