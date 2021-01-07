-- We want POST methods with form values "username" and "password" set
wrk.method = "POST"
wrk.headers["Content-Type"] = "application/x-www-form-urlencoded"
--wrk.body = "username=kendrick&password=password"
--requests = {}
ptr = 0
--NUM_USERS = 200
--
--init = function(args)
--    for i = 1, NUM_USERS, 1 do
--        body = string.format("username=%s&password=%s", tostring(i), "password")
--        requests[i] = wrk.format("POST", "/login", nil, body)
--    end
--    ptr = math.random(1, NUM_USERS)
--    print("Generated requests table\n")
--end
--
request = function()
    ptr = (ptr + 1) % 200
    return wrk.format(nil, "/login", nil, string.format("username=%s&password=password", tostring(ptr)))
end
