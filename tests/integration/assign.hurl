POST {{base_url}}/register
Content-Type: application/json
{"email":"assign-owner-20260614@example.com","password":"secure123","name":"Assign Owner","team_id":"a0000000-0000-0000-0000-000000000001"}
HTTP 201
[Captures]
owner_token: jsonpath "$.token"
owner_id: jsonpath "$.user.id"

POST {{base_url}}/register
Content-Type: application/json
{"email":"assign-user2-20260614@example.com","password":"secure123","name":"Assign User 2","team_id":"a0000000-0000-0000-0000-000000000001"}
HTTP 201
[Captures]
user2_id: jsonpath "$.user.id"

POST {{base_url}}/register
Content-Type: application/json
{"email":"assign-out-20260614@example.com","password":"secure123","name":"Assign Outsider","team_id":"b0000000-0000-0000-0000-000000000002"}
HTTP 201
[Captures]
outsider_id: jsonpath "$.user.id"

POST {{base_url}}/tasks
Authorization: Bearer {{owner_token}}
Idempotency-Key: f0000001-0000-0000-0000-000000000001
Content-Type: application/json
{"title":"Task to Assign"}
HTTP 201
[Captures]
assign_task_id: jsonpath "$.data.id"

POST {{base_url}}/tasks/{{assign_task_id}}/assign
Authorization: Bearer {{owner_token}}
Content-Type: application/json
{"user_id":"{{user2_id}}"}
HTTP 200
[Asserts]
jsonpath "$.data.assignee_id" == "{{user2_id}}"

POST {{base_url}}/tasks/{{assign_task_id}}/assign
Authorization: Bearer {{owner_token}}
Content-Type: application/json
{"user_id":"{{outsider_id}}"}
HTTP 403
[Asserts]
jsonpath "$.code" == "ERR_FORBIDDEN"
