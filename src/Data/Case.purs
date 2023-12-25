module Dagobert.Data.Case where

import Dagobert.Data.Common (Common', common')
import Record (merge)

type Case =
  { id             :: Int
  , name           :: String
  , closed         :: Boolean
  , classification :: String
  , severity       :: String
  , outcome        :: String
  , summary        :: String
  | Common'
  }

type CaseStub =
  { id             :: Int
  , name           :: String
  , closed         :: Boolean
  , classification :: String
  , severity       :: String
  , outcome        :: String
  , summary        :: String
  }

newCase :: Case
newCase = merge common' { id: 0, name: "", closed: false, classification: "", severity: "", outcome: "", summary: "" }

caseSeverities :: Array String
caseSeverities = [ "Low", "Medium", "High" ]

caseOutcomes :: Array String
caseOutcomes = [ "False positive", "True positive", "Benign positive" ]