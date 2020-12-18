-- We want POST methods with form values "username" and "password" set
wrk.method = "POST"
wrk.headers["Content-Type"] = "application/x-www-form-urlencoded"
requests = {}

init = function(args)
    for i = 1, 10000, 1 do
        body = string.format("username=%s&password=%s", tostring(i), "password")
        requests[i] = wrk.format("POST", "/login", nil, body)
    end
    print("Generated requests table\n")
    --print(table.concat(requests, "\n"))
end

request = function()
    idx = math.random(1, 200)
    return requests[idx]
end
