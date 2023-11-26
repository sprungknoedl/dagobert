module Dagobert.Data.Evidence where

type Evidence =
  { id          :: Int
  , type        :: String
  , name        :: String
  , description :: String
  , hash        :: String
  , location    :: String
  }

newEvidence :: Evidence
newEvidence = {id: 0, type: "", name: "", description: "", hash: "", location: ""}

evidenceTypes :: Array String
evidenceTypes = ["File", "Log", "Artifacts Collection", "System Image", "Memory Dump", "Other"]