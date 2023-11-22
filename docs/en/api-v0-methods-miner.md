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
  * [ListBlocks](#ListBlocks)
* [Log](#Log)
  * [LogList](#LogList)
  * [LogSetLevel](#LogSetLevel)
* [Query](#Query)
  * [QueryRecord](#QueryRecord)
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

### ListBlocks


Perms: read

Inputs:
```json
[
  {
    "Miners": [
      "t01234"
    ],
    "Limit": 42,
    "Offset": 42
  }
]
```

Response:
```json
[
  {
    "ParentEpoch": 0,
    "ParentKey": "",
    "Epoch": 9,
    "Miner": "string value",
    "Cid": "string value",
    "WinningAt": "0001-01-01T00:00:00Z",
    "MineState": 0,
    "Consuming": 9
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

## Query


### QueryRecord


Perms: read

Inputs:
```json
[
  {
    "Miner": "t01234",
    "Epoch": 10101,
    "Limit": 42
  }
]
```

Response:
```json
[
  {}
]
```

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

