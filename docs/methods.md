
## `GetAssets`
### Body
```
{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "GetAssets"
}
```
### Parameters
_none_
### Response
```
{
  "scids": [ 
      "a957231ba28b6b72bb361cad75f15f684f4cd3ef3e1e8986261bc82d20625cd8",
      "9054fb4fa91289814336009f707881b6b99202b64d7cb1f9c589a66613a5149e",
      "ad2e7b37c380cc1aed3a6b27224ddfc92a2d15962ca1f4d35e530dba0f9575a9"
    ]
}
```

## `GetAssetBalance`
### Body
```
{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "GetAssetBalance"
    "params": {
      "scid": "86acadbda70bbaf25b03425a84612072db03fe8488837b534a1d6049833490fc"
      "height": -1,
  }
}
```
### Parameters
- SCID - required
- Height - required
> Use `-1` for current topo height

### Response
```
{
  "balance": 123456
}
```
## `GetTXEstimate`
### Body
```
{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "GetTXEstimate"
    "params": {
      "transfers": [
        {
          "scid": "4f3a9c2b1e0d8a7c6b5a4d3e2f1a9b8c7d6e5f4a3b2c1d0e9f8a7b6c5d4",
          "destination": "dero1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqg",
          "amount": 2500000000,
          "burn": 0,
          "payload_rpc": [
            {
              "name": "entrypoint",
              "datatype": "S",
              "value": "FUNCTION_NAME"
            },
            {
              "name": "function_arg",
              "datatype": "S",
              "value": "Test transfer"
            }
          ]
        }
      ],
      "sc": "MySmartContract",
      "sc_value": 1000000,
      "scid": "8a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d",
      "sc_rpc": [
        {
          "name": "entrypoint",
          "datatype": "S",
          "value": "FUNCTION_NAME"
        },
        {
          "name": "function_arg",
          "datatype": "S",
          "value": "Test transfer"
        }
      ],
      "ringsize": 16,
      "signer": "dero1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqg"
    }
}
```
### Parameters
_none_
> N.B. Build a transfer, get a fees estimate.
### Response
```
{
  "fees": 123456
}
```
### Parameters
- SCID - required
- Height - required
> Use `-1` for current topo height

### Response
```
{
  "balance": 123456
}
```

## `AttemptEPOCHWithAddr`
### Body
```
{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "GetAssetBalance"
    "params": {
      "address": "dero1qyvqpdftj8r6005xs20rnflakmwa5pdxg9vcjzdcuywq2t8skqhvwqglt6x0g"
      "hashes": 1000,
  }
}
```
### Parameters
- Address - required
- Hashes - required

### Response
```
{
  "epochDuration":580, 
  "epochHashPerSecond":1721.21, 
  "epochHashes":1000, 
  "epochSubmitted":0
}
```
## `Gnomon.GetAllOwnersAndSCIDs`
### Body
```
{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "Gnomon.GetAllOwnersAndSCIDs"
    "params": {
      "tag": "all"
  }
}
```
### Parameters
- tag - optional, if left blank defaults to all
> (NOTE: this feature is subject to change as it relies on upstream tool) 


### Response
```
{
  "allOwners":[ 
    fffe1bb8098646c13a03467dfe0581f292cb00559ac3dffc78a7430397c25aef:dero1qyq6p3cwu905q7urmdkmh67p6ceqh7kmaczvhfdmz8scfxf4j3kjgqqnvwfmj,
    fffee4409d59e71bb48fabcfd3b2b64af49f2489b65c646d86a46dc8b4dedda2:dero1qy429gdgtwzz07pslkf8q3fd47lp75r6vc9qth79t8vmja5nxzvpjqqftz6n4
    ...
  ], 
}
```
## `Gnomon.GetAllSCIDVariableDetails`
### Body
```
{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "Gnomon.GetAllSCIDVariableDetails"
    "params": {
      "tag": "all"
      "scid": "fffee4409d59e71bb48fabcfd3b2b64af49f2489b65c646d86a46dc8b4dedda2"
  }
}
```
### Parameters
- scid - required
- tag - optional, if left blank defaults to all
> (NOTE: this feature is subject to change as it relies on upstream tool) 


### Response
```
{
  "allVariables": [
    {
      "Key": "type",
      "Value": "G45-NFT"
    },
    {
      "Key": "C",
      "Value": "Function InitializePrivate(collection String, metadataFormat String, metadata String) Uint64\n1 IF EXISTS(\"minter\") == 1 THEN GOTO 11\n2 STORE(\"minter\", SIGNER())\n3 STORE(\"type\", \"G45-NFT\")\n4 STORE(\"owner\", \"\")\n5 STORE(\"timestamp\", BLOCK_TIMESTAMP())\n6 SEND_ASSET_TO_ADDRESS(SIGNER(), 1, SCID())\n7 STORE(\"collection\", collection)\n8 STORE(\"metadataFormat\", metadataFormat)\n9 STORE(\"metadata\", metadata)\n10 RETURN 0\n11 RETURN 1\nEnd Function\n\nFunction DisplayNFT() Uint64\n1 IF ADDRESS_STRING(SIGNER()) == \"\" THEN GOTO 5\n2 IF ASSETVALUE(SCID()) != 1 THEN GOTO 5\n3 STORE(\"owner\", ADDRESS_STRING(SIGNER()))\n4 RETURN 0\n5 RETURN 1\nEnd Function\n\nFunction RetrieveNFT() Uint64\n1 IF LOAD(\"owner\") != ADDRESS_STRING(SIGNER()) THEN GOTO 5\n2 SEND_ASSET_TO_ADDRESS(SIGNER(), 1, SCID())\n3 STORE(\"owner\", \"\")\n4 RETURN 0\n5 RETURN 1\nEnd Function"
    },
    {
      "Key": "collection",
      "Value": "92c8e4dbab3f9f3245be688eaa8d9456e660ad6031901a0c214988d5d29c2acf"
    },
    {
      "Key": "metadata",
      "Value": {
        "attributes": {
          "City": "Singapore",
          "Country": "Singapore",
          "Landmark": "Botanic Gardens"
        },
        "id": 48,
        "image": "ipfs://QmfPXr65DVgECH5TRn6c7pWwgBWzWVyH7oJNYwC3PBVc8j/DerBnB%20%2348.jpg",
        "name": "DerBNB #48"
      }
    },
    {
      "Key": "metadataFormat",
      "Value": "json"
    },
    {
      "Key": "minter",
      "Value": "dero1qy3gr5zgxqlwtcaa03uvplczrrgaa8w6fagjpvsngc69hu884jedjqqj20tn6"
    },
    {
      "Key": "owner",
      "Value": "dero1qy429gdgtwzz07pslkf8q3fd47lp75r6vc9qth79t8vmja5nxzvpjqqftz6n4"
    },
    {
      "Key": "timestamp",
      "Value": 1684239153
    }
  ], 
}
```
