module Dagobert.Data.Indicator where

type Indicator =
  { id          :: Int
  , type        :: String
  , value       :: String
  , tlp         :: String
  , description :: String
  , source      :: String
  }

newIndicator :: Indicator
newIndicator = { id: 0, type: "", value: "", tlp: "", description: "", source: ""}

mkIndicator :: Int -> String -> String -> String -> String -> String -> Indicator
mkIndicator = { id: _, type: _, value: _, tlp: _, description: _, source: _}

indicatorTypes :: Array String
indicatorTypes = ["IP", "Domain", "URL", "Path", "Hash", "Service", "Other"]

tlpValues :: Array String
tlpValues = ["TLP:RED", "TLP:AMBER", "TLP:GREEN", "TLP:CLEAR"]
