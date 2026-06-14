POST {{base_url}}/register
Content-Type: application/json
{"email":"idem-test-20260614@example.com","password":"secure123","name":"Idem Test"}
HTTP 201
[Captures]
token: jsonpath "$.token"

POST {{base_url}}/tasks
Authorization: Bearer {{token}}
Idempotency-Key: 11111111-1111-1111-1111-111111111111
Content-Type: application/json
{"title":"Idempotent Task"}
HTTP 201
[Captures]
idem_task_id: jsonpath "$.data.id"
[Asserts]
jsonpath "$.data.title" == "Idempotent Task"

POST {{base_url}}/tasks
Authorization: Bearer {{token}}
Idempotency-Key: 11111111-1111-1111-1111-111111111111
Content-Type: application/json
{"title":"Idempotent Task"}
HTTP 201
[Asserts]
jsonpath "$.data.title" == "Idempotent Task"
jsonpath "$.data.id" == "{{idem_task_id}}"
