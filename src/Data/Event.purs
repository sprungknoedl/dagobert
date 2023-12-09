module Dagobert.Data.Event where

import Dagobert.Data.Common (Common, common)
import Record (merge)

type Event =
  { id           :: Int
  , time         :: String
  , type         :: String
  , assetA       :: String 
  , assetB       :: String
  , direction    :: String
  , event        :: String
  , raw          :: String
  | Common
  }

type EventStub =
  { id           :: Int
  , time         :: String
  , type         :: String
  , assetA       :: String 
  , assetB       :: String
  , direction    :: String
  , event        :: String
  , raw          :: String
  }

newEvent :: Event
newEvent = merge common { id: 0, time: "", type: "", assetA: "", assetB: "", direction: "", event: "", raw: "" }

eventTypes :: Array String
eventTypes = ["Event Log", "File", "Human", "Lateral Movement", "Exfiltration", "Malware", "C2", "DFIR", "Other"]

directionValues :: Array String
directionValues = ["", "→", "←"]