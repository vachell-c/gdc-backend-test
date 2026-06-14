POST {{base_url}}/register
Content-Type: application/json
{"email":"auth-test-20260614@example.com","password":"secure123","name":"Auth Test"}
HTTP 201
[Captures]
token: jsonpath "$.token"
[Asserts]
jsonpath "$.token" exists
jsonpath "$.user.id" exists

POST {{base_url}}/login
Content-Type: application/json
{"email":"auth-test-20260614@example.com","password":"secure123"}
HTTP 200
[Captures]
login_token: jsonpath "$.token"
[Asserts]
jsonpath "$.token" exists

POST {{base_url}}/login
Content-Type: application/json
{"email":"auth-test-20260614@example.com","password":"wrongpassword"}
HTTP 401
[Asserts]
jsonpath "$.status" == 401
jsonpath "$.code" == "ERR_UNAUTHORIZED"
