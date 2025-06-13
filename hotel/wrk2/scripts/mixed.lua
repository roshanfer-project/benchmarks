local url1 = "http://192.168.1.100:2000/reservation?inDate=2025-05-20&outDate=2025-05-22&hotelId=4&customerName=Alice&username=Cornell_1&password=1111111111&number=1"
local url2 = "http://192.168.1.100:2000/hotels?lat=37.7867&lon=-122.4112&inDate=2024-08-15&outDate=2024-08-17"

function init(args)
  -- probability for url1
  p = tonumber(args[1]) or 0.7
  -- seed per thread
  math.randomseed(os.time() + tonumber(tostring({}):sub(7)))
end

function request()
  local target = math.random() < p and url1 or url2
  -- default method is GET; to change, e.g. POST:
  -- return wrk.format("POST", target, headers, body)
  local headers = {}
  return wrk.format("GET", target, headers, nil)
end
