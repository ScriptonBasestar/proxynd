-- Analytics Dashboard Load Test
-- Simulates dashboard loading with multiple concurrent requests

local token = os.getenv("ENTERPRISE_TOKEN") or ""
local counter = 0

-- Dashboard endpoints
local dashboard_endpoints = {
    "/api/v1/enterprise/analytics/overview",
    "/api/v1/enterprise/analytics/usage",
    "/api/v1/enterprise/analytics/performance",
    "/api/v1/enterprise/analytics/cache-efficiency",
    "/api/v1/enterprise/analytics/trends"
}

function request()
    counter = counter + 1

    -- Cycle through dashboard endpoints
    local endpoint = dashboard_endpoints[(counter % #dashboard_endpoints) + 1]

    local headers = {}
    headers["Authorization"] = "Bearer " .. token
    headers["Accept"] = "application/json"

    return wrk.format("GET", endpoint, headers, nil)
end

function done(summary, latency, requests)
    io.write("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
    io.write("Analytics Dashboard Load Test Results\n")
    io.write("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

    io.write(string.format("Dashboard Loads:   %d\n", summary.requests / #dashboard_endpoints))
    io.write(string.format("Requests/sec:      %.2f\n", summary.requests / (summary.duration / 1000000)))
    io.write(string.format("Avg Latency:       %.2fms\n", latency.mean / 1000))
    io.write(string.format("P95 Latency:       %.2fms\n", latency:percentile(95) / 1000))

    -- Dashboard loading performance verdict
    local avg_latency_ms = latency.mean / 1000
    io.write("\nDashboard Performance: ")
    if avg_latency_ms < 200 then
        io.write("EXCELLENT ✓\n")
    elseif avg_latency_ms < 500 then
        io.write("GOOD ✓\n")
    elseif avg_latency_ms < 1000 then
        io.write("ACCEPTABLE ⚠\n")
    else
        io.write("NEEDS OPTIMIZATION ✗\n")
    end
    io.write("\n")
end
