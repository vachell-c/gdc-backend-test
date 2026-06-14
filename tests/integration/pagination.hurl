POST {{base_url}}/register
Content-Type: application/json
{"email":"page-test-20260614@example.com","password":"secure123","name":"Page Test"}
HTTP 201
[Captures]
token: jsonpath "$.token"

POST {{base_url}}/tasks
Authorization: Bearer {{token}}
Idempotency-Key: a0000001-0000-0000-0000-000000000001
Content-Type: application/json
{"title":"Task Alpha","status":"pending"}
HTTP 201

POST {{base_url}}/tasks
Authorization: Bearer {{token}}
Idempotency-Key: b0000002-0000-0000-0000-000000000002
Content-Type: application/json
{"title":"Task Beta","status":"completed"}
HTTP 201

POST {{base_url}}/tasks
Authorization: Bearer {{token}}
Idempotency-Key: c0000003-0000-0000-0000-000000000003
Content-Type: application/json
{"title":"Task Gamma","status":"in_progress"}
HTTP 201

POST {{base_url}}/tasks
Authorization: Bearer {{token}}
Idempotency-Key: d0000004-0000-0000-0000-000000000004
Content-Type: application/json
{"title":"Task Delta","status":"completed"}
HTTP 201

POST {{base_url}}/tasks
Authorization: Bearer {{token}}
Idempotency-Key: e0000005-0000-0000-0000-000000000005
Content-Type: application/json
{"title":"Task Epsilon","status":"pending"}
HTTP 201

GET {{base_url}}/tasks?status=completed
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.data" count >= 1
jsonpath "$.meta.total" > 0

GET {{base_url}}/tasks?limit=2&page=1
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.data" count <= 2
jsonpath "$.meta.page" == 1
jsonpath "$.meta.limit" == 2
jsonpath "$.meta.total" >= 1

GET {{base_url}}/tasks?search=Alpha
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.meta.total" > 0
