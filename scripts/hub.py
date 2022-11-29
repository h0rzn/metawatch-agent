#!/usr/bin/python3

import asyncio
import json
import websockets
import time
import requests
from sys import exit

until_usub = 5

sub = {
  "container_id": "76a659815abee87bd511219ca08df6f7107b83e69f766726a04adc38a72f64e4",
  "event": "subscribe",
  "type": "logs"
}

usub = {
  "container_id": "76a659815abee87bd511219ca08df6f7107b83e69f766726a04adc38a72f64e4",
  "event": "unsubscribe",
  "type": "logs"
}


async def run():
  async with websockets.connect("ws://localhost:8080/test") as ws:
    print("[SND]", json.dumps(sub))
    await ws.send(json.dumps(sub))
    
    start_time = time.time()
    while True:
      cur_time = time.time()
      elap_time = cur_time - start_time

      if elap_time > until_usub:
        print("\n[USUB]\n")
        await ws.send(json.dumps(usub))
        break
    
      res = await ws.recv()
      print("[RCV]", res)



  
# r = requests.get("http://localhost:8080/containers/all")
# print(r.status_code)
# if not r.ok:
#   print("error", r.status_code)
#   exit()

# for container in range(r.json()):
#   print(container.id)

asyncio.get_event_loop().run_until_complete(run())