module Dagobert.Data.Asset where

import Dagobert.Data.Common (Common, common)
import Record (merge)

type Asset =
  { id          :: Int
  , type        :: String
  , name        :: String
  , ip          :: String
  , description :: String
  , compromised :: String
  , analysed    :: Boolean
  | Common
  }

type AssetStub =
  { id          :: Int
  , type        :: String
  , name        :: String
  , ip          :: String
  , description :: String
  , compromised :: String
  , analysed    :: Boolean
  }

newAsset :: Asset
newAsset = merge common { id: 0, type: "", name: "", ip: "", description: "", compromised: "", analysed: false }

assetTypes :: Array String
assetTypes = [ "Account", "Desktop", "Server", "Other" ]

compromiseStates :: Array String
compromiseStates = [ "Compromised", "Not compromised", "Unknown" ]