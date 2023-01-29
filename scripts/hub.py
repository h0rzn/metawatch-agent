#!/usr/bin/python3

import asyncio
import json
import websockets
import time
import requests
from sys import exit
from datetime import datetime

until_usub = 5
CID = "fdaaaa9dcace802715dbb865eb784bf6b8aa48de9d8a425ca11a472edc72d240"
TYPE = "logs"

sub = {
  "container_id": CID,
  "event": "subscribe",
  "type": TYPE
}

usub = {
  "container_id": CID,
  "event": "unsubscribe",
  "type": TYPE
}



async def run():
  async with websockets.connect("ws://localhost:8080/stream") as ws:
    print("[SND]", json.dumps(sub))
    await ws.send(json.dumps(sub))
    
    start_time = time.time()
    while True:
      cur_time = time.time()
      elap_time = cur_time - start_time

      res = await ws.recv()

      cur = datetime.now().strftime("%H:%M:%S")
      print("RCV", cur, res)

      if elap_time > until_usub:
        print("\n[USUB]\n")
        await ws.send(json.dumps(usub))

        # time.sleep(5)
        # print("woke up: resubscribing")
        # await ws.send(json.dumps(sub))

        # while True:
        #   res = await ws.recv()
        #   print("RCV", res)

      

# async def run():
#   async with websockets.connect("ws://localhost:8080/stream") as ws:
#     print("[SND]", json.dumps(sub_comb))
#     await ws.send(json.dumps(sub_comb))
    
#     while True:

#       res = await ws.recv()

#       print("RCV", res)

    



  
# r = requests.get("http://localhost:8080/containers/all")
# print(r.status_code)
# if not r.ok:
#   print("error", r.status_code)
#   exit()

# for container in range(r.json()):
#   print(container.id)

asyncio.get_event_loop().run_until_complete(run())
