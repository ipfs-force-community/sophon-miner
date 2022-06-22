# Groups
* [](#)
  * [Closing](#Closing)
  * [Session](#Session)
  * [Shutdown](#Shutdown)
  * [Start](#Start)
  * [Stop](#Stop)
  * [Version](#Version)
* [Add](#Add)
  * [AddAddress](#AddAddress)
* [Auth](#Auth)
  * [AuthNew](#AuthNew)
  * [AuthVerify](#AuthVerify)
* [Count](#Count)
  * [CountWinners](#CountWinners)
* [List](#List)
  * [ListAddress](#ListAddress)
* [Log](#Log)
  * [LogList](#LogList)
  * [LogSetLevel](#LogSetLevel)
* [States](#States)
  * [StatesForMining](#StatesForMining)
* [Update](#Update)
  * [UpdateAddress](#UpdateAddress)
## 


### Closing


Perms: read

Inputs: `null`

Response: `{}`

### Session


Perms: read

Inputs: `null`

Response: `"07070707-0707-0707-0707-070707070707"`

### Shutdown


Perms: admin

Inputs: `null`

Response: `{}`

### Start


Perms: admin

Inputs:
```json
[
  [
    "t01234"
  ]
]
```

Response: `{}`

### Stop


Perms: admin

Inputs:
```json
[
  [
    "t01234"
  ]
]
```

Response: `{}`

### Version


Perms: read

Inputs: `null`

Response:
```json
{
  "Version": "string value",
  "APIVersion": 0,
  "BlockDelay": 0
}
```

## Add


### AddAddress


Perms: admin

Inputs:
```json
[
  {
    "Addr": "t01234",
    "Id": "string value",
    "Name": "string value"
  }
]
```

Response: `{}`

## Auth


### AuthNew


Perms: admin

Inputs:
```json
[
  [
    "write"
  ]
]
```

Response: `"Ynl0ZSBhcnJheQ=="`

### AuthVerify


Perms: read

Inputs:
```json
[
  "string value"
]
```

Response:
```json
[
  "write"
]
```

## Count


### CountWinners


Perms: read

Inputs:
```json
[
  [
    "t01234"
  ],
  10101,
  10101
]
```

Response:
```json
[
  {
    "miner": "t01234",
    "totalWinCount": 0,
    "msg": "string value",
    "winEpochList": null
  }
]
```

## List


### ListAddress


Perms: read

Inputs: `null`

Response:
```json
[
  {
    "Addr": "t01234",
    "Id": "string value",
    "Name": "string value"
  }
]
```

## Log


### LogList


Perms: write

Inputs: `null`

Response:
```json
[
  "string value"
]
```

### LogSetLevel


Perms: write

Inputs:
```json
[
  "string value",
  "string value"
]
```

Response: `{}`

## States


### StatesForMining


Perms: read

Inputs:
```json
[
  [
    "t01234"
  ]
]
```

Response:
```json
[
  {
    "Addr": "t01234",
    "IsMining": false,
    "Err": [
      "string value"
    ]
  }
]
```

## Update


### UpdateAddress


Perms: write

Inputs:
```json
[
  9,
  9
]
```

Response:
```json
[
  {
    "Addr": "t01234",
    "Id": "string value",
    "Name": "string value"
  }
]
```

