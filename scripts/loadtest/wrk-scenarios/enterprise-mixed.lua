-- Mixed Enterprise API Load Test Scenario
-- Tests multiple endpoints with realistic traffic distribution

local token = os.getenv("ENTERPRISE_TOKEN") or ""

-- Endpoint distribution (weighted)
local endpoints = {
    {path = "/api/v1/enterprise/analytics/overview", weight = 30},
    {path = "/api/v1/enterprise/rbac/roles", weight = 20},
    {path = "/api/v1/enterprise/audit/events", weight = 25},
    {path = "/api/v1/enterprise/security/vulnerabilities", weight = 15},
    {path = "/api/v1/enterprise/alerts", weight = 5},
    {path = "/api/v1/enterprise/license/info", weight = 5}
}

-- Calculate cumulative weights
local cumulative = 0
for i, endpoint in ipairs(endpoints) do
    cumulative = cumulative + endpoint.weight
    endpoint.cumulative = cumulative
end
local total_weight = cumulative

-- Request counter
local counter = 0

function request()
    counter = counter + 1

    -- Select endpoint based on weight
    local rand = math.random(1, total_weight)
    local selected_path = endpoints[1].path
    for i, endpoint in ipairs(endpoints) do
        if rand <= endpoint.cumulative then
            selected_path = endpoint.path
            break
        end
    end

    -- Build request
    local headers = {}
    headers["Authorization"] = "Bearer " .. token
    headers["Accept"] = "application/json"

    return wrk.format("GET", selected_path, headers, nil)
end

function response(status, headers, body)
    if status ~= 200 then
        print("Error: HTTP " .. status)
    end
end

function done(summary, latency, requests)
    io.write("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
    io.write("Mixed Enterprise API Load Test Results\n")
    io.write("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

    io.write(string.format("Total Requests:    %d\n", summary.requests))
    io.write(string.format("Total Duration:    %.2fs\n", summary.duration / 1000000))
    io.write(string.format("Requests/sec:      %.2f\n", summary.requests / (summary.duration / 1000000)))
    io.write(string.format("Avg Latency:       %.2fms\n", latency.mean / 1000))
    io.write(string.format("P50 Latency:       %.2fms\n", latency:percentile(50) / 1000))
    io.write(string.format("P95 Latency:       %.2fms\n", latency:percentile(95) / 1000))
    io.write(string.format("P99 Latency:       %.2fms\n", latency:percentile(99) / 1000))
    io.write(string.format("Max Latency:       %.2fms\n", latency.max / 1000))
    io.write(string.format("Errors:            %d (%.2f%%)\n",
        summary.errors.connect + summary.errors.read + summary.errors.write + summary.errors.status + summary.errors.timeout,
        (summary.errors.connect + summary.errors.read + summary.errors.write + summary.errors.status + summary.errors.timeout) / summary.requests * 100))

    io.write("\n")
end
