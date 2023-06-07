# Groups
* [](#)
  * [Closing](#Closing)
  * [Session](#Session)
  * [Shutdown](#Shutdown)
  * [Start](#Start)
  * [Stop](#Stop)
  * [Version](#Version)
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
* [Warmup](#Warmup)
  * [WarmupForMiner](#WarmupForMiner)
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


Perms: write

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


Perms: write

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
  "APIVersion": 0
}
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
    "Name": "string value",
    "OpenMining": false
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


Perms: admin

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
    "Name": "string value",
    "OpenMining": false
  }
]
```

## Warmup


### WarmupForMiner


Perms: write

Inputs:
```json
[
  "t01234"
]
```

Response: `{}`

