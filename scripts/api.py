import requests
import json

TOKEN = ""
MONGO_CID = "e63d12ed9e74cb2d5994e9e0356aad7588939108fb2a1e5ec729274e07e820bf"
PORT = "8081"

def h(): 
    return {
        "Authorization": "Bearer " + TOKEN,
        "Content-Type": "application/json"
    }

def login():
    p = {
        "username": "master",
        "password": "master"
    }
    r = requests.post("http://localhost:"+PORT+"/login", params=p)
    print(r.status_code)
    if r.ok:
        return True, r.json()["token"]
    return False, ""

def metrics(amount):
    h = {
        "Authorization": "Bearer " + TOKEN,
        "Content-Type": "application/json"
    }
    p = {
        # "from": "2022-12-15T13:00:00Z",
        "from": "2023-01-10T18:00:00Z",
        # "to": "2022-12-15T13:30:00Z"
        "to": "2023-01-10T18:05:00Z",
        "amount": amount
    }
    r = requests.get("http://localhost:"+PORT+"/api/containers/e63d12ed9e74cb2d5994e9e0356aad7588939108fb2a1e5ec729274e07e820bf/metrics", headers=h, params=p)
    print(json.dumps(r.json(), indent=4))

def user_add(name):
    d = {
        "name": "123456",
        "password": "123"
    }
    r = requests.post("http://localhost:"+PORT+"/api/users", json=d)
    print(r.status_code)
    print(r.text)

def users_get():
    r = requests.get("http://localhost:"+PORT+"/api/users", headers=h())
    print(r.status_code)
    print(r.text)


def user_update(id):
    update = {
        "name1": "newly patched name"
    }

    r = requests.patch("http://localhost:"+PORT+"/api/users/"+id, json=update)
    print(r.status_code)
    print(r.text)

if __name__ == "__main__":
    status, token = login()
    if status:
        TOKEN = token
    else:
        print("auth failed")

    #user_add("myname")
    users_get()
    # user_update("63d815c8468c1acb0dd64709")
