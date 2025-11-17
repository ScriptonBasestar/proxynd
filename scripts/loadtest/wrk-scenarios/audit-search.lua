-- Audit Search Load Test
-- Tests audit logging and search performance

local token = os.getenv("ENTERPRISE_TOKEN") or ""
local counter = 0

-- Audit search patterns
local search_queries = {
    "?page=1&per_page=20",
    "?page=1&per_page=50",
    "?page=2&per_page=20",
    "/search?query=login",
    "/search?query=create&user_id=user_123",
    "/search?action=delete"
}

function request()
    counter = counter + 1

    -- Rotate through search queries
    local query = search_queries[(counter % #search_queries) + 1]
    local path = "/api/v1/enterprise/audit/events" .. query

    local headers = {}
    headers["Authorization"] = "Bearer " .. token
    headers["Accept"] = "application/json"

    return wrk.format("GET", path, headers, nil)
end

function done(summary, latency, requests)
    io.write("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
    io.write("Audit Search Load Test Results\n")
    io.write("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

    io.write(string.format("Total Searches:    %d\n", summary.requests))
    io.write(string.format("Searches/sec:      %.2f\n", summary.requests / (summary.duration / 1000000)))
    io.write(string.format("Avg Latency:       %.2fms\n", latency.mean / 1000))
    io.write(string.format("P95 Latency:       %.2fms\n", latency:percentile(95) / 1000))
    io.write("\n")
end
