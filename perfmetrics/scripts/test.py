import time
from time import sleep

start_time = time.time()
sleep(18005)
end_time = time.time()

print("Timedout")

print(end_time - start_time)
