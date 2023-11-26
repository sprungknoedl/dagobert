module Dagobert.Data.Asset where

type Asset =
  { id          :: Int
  , type        :: String
  , name        :: String
  , ip          :: String
  , description :: String
  , compromised :: String
  , analysed    :: Boolean
  }

newAsset :: Asset
newAsset = { id: 0, type: "", name: "", ip: "", description: "", compromised: "", analysed: false }

assetTypes :: Array String
assetTypes = [ "Account", "Desktop", "Server", "Other" ]

compromiseStates :: Array String
compromiseStates = [ "Compromised", "Not compromised", "Unknown" ]