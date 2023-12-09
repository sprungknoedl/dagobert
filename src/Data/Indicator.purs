module Dagobert.Data.Indicator where

import Dagobert.Data.Common (Common, common)
import Record (merge)

type Indicator =
  { id          :: Int
  , type        :: String
  , value       :: String
  , tlp         :: String
  , description :: String
  , source      :: String
  | Common
  }

type IndicatorStub =
  { id          :: Int
  , type        :: String
  , value       :: String
  , tlp         :: String
  , description :: String
  , source      :: String
  }

newIndicator :: Indicator
newIndicator = merge common { id: 0, type: "", value: "", tlp: "", description: "", source: "" }

indicatorTypes :: Array String
indicatorTypes = ["IP", "Domain", "URL", "Path", "Hash", "Service", "Other"]

tlpValues :: Array String
tlpValues = ["TLP:RED", "TLP:AMBER", "TLP:GREEN", "TLP:CLEAR"]
