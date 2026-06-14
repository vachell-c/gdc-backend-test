POST {{base_url}}/register
Content-Type: application/json
{"email":"tasks-test-20260614@example.com","password":"secure123","name":"Tasks Test"}
HTTP 201
[Captures]
token: jsonpath "$.token"

POST {{base_url}}/tasks
Authorization: Bearer {{token}}
Idempotency-Key: c0000001-0000-0000-0000-000000000001
Content-Type: application/json
{"title":"Test Task","description":"Integration test","status":"pending"}
HTTP 201
[Captures]
task_id: jsonpath "$.data.id"
[Asserts]
jsonpath "$.data.title" == "Test Task"

GET {{base_url}}/tasks
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.data" count >= 1

GET {{base_url}}/tasks/{{task_id}}
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.data.id" == "{{task_id}}"

PUT {{base_url}}/tasks/{{task_id}}
Authorization: Bearer {{token}}
Content-Type: application/json
{"title":"Updated Task","status":"in_progress"}
HTTP 200
[Asserts]
jsonpath "$.data.title" == "Updated Task"

DELETE {{base_url}}/tasks/{{task_id}}
Authorization: Bearer {{token}}
HTTP 204

GET {{base_url}}/tasks/{{task_id}}
Authorization: Bearer {{token}}
HTTP 404
[Asserts]
jsonpath "$.status" == 404
