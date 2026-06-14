POST {{base_url}}/register
Content-Type: application/json
{"email":"errors-test-20260614@example.com","password":"secure123","name":"Errors Test"}
HTTP 201
[Captures]
token: jsonpath "$.token"

POST {{base_url}}/tasks
Authorization: Bearer {{token}}
Idempotency-Key: e0000001-0000-0000-0000-000000000001
Content-Type: application/json
{"title":""}
HTTP 400
[Asserts]
jsonpath "$.status" == 400
jsonpath "$.code" == "ERR_VALIDATION"

GET {{base_url}}/tasks
HTTP 401
[Asserts]
jsonpath "$.status" == 401

GET {{base_url}}/tasks/00000000-0000-0000-0000-000000000000
Authorization: Bearer {{token}}
HTTP 404
[Asserts]
jsonpath "$.status" == 404
jsonpath "$.code" == "ERR_NOT_FOUND"
