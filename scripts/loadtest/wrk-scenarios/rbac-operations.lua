-- RBAC Operations Load Test
-- Simulates realistic RBAC management operations

local token = os.getenv("ENTERPRISE_TOKEN") or ""
local counter = 0

-- RBAC operations with weights
local operations = {
    {method = "GET", path = "/api/v1/enterprise/rbac/roles", weight = 40},
    {method = "GET", path = "/api/v1/enterprise/rbac/roles/role_admin", weight = 20},
    {method = "GET", path = "/api/v1/enterprise/rbac/permissions", weight = 20},
    {method = "GET", path = "/api/v1/enterprise/rbac/users/user_123/roles", weight = 15},
    {method = "POST", path = "/api/v1/enterprise/rbac/roles", weight = 5,
     body = '{"name":"test_role","description":"Test","permissions":["read:packages"]}'}
}

-- Calculate cumulative weights
local cumulative = 0
for i, op in ipairs(operations) do
    cumulative = cumulative + op.weight
    op.cumulative = cumulative
end
local total_weight = cumulative

function request()
    counter = counter + 1

    -- Select operation based on weight
    local rand = math.random(1, total_weight)
    local selected_op = operations[1]
    for i, op in ipairs(operations) do
        if rand <= op.cumulative then
            selected_op = op
            break
        end
    end

    -- Build request
    local headers = {}
    headers["Authorization"] = "Bearer " .. token
    headers["Accept"] = "application/json"
    if selected_op.body then
        headers["Content-Type"] = "application/json"
    end

    return wrk.format(selected_op.method, selected_op.path, headers, selected_op.body)
end

function done(summary, latency, requests)
    io.write("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
    io.write("RBAC Operations Load Test Results\n")
    io.write("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

    io.write(string.format("Total Requests:    %d\n", summary.requests))
    io.write(string.format("Requests/sec:      %.2f\n", summary.requests / (summary.duration / 1000000)))
    io.write(string.format("P95 Latency:       %.2fms\n", latency:percentile(95) / 1000))
    io.write("\n")
end
