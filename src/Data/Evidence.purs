module Dagobert.Data.Evidence where

import Dagobert.Data.Common (Common, common)
import Record (merge)

type Evidence =
  { id           :: Int
  , type         :: String
  , name         :: String
  , description  :: String
  , hash         :: String
  , location     :: String
  | Common
  }

type EvidenceStub =
  { id           :: Int
  , type         :: String
  , name         :: String
  , description  :: String
  , hash         :: String
  , location     :: String
  }

newEvidence :: Evidence
newEvidence = merge common { id: 0, type: "", name: "", description: "", hash: "", location: "" }

evidenceTypes :: Array String
evidenceTypes = ["File", "Log", "Artifacts Collection", "System Image", "Memory Dump", "Other"]