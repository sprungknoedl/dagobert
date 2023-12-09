module Dagobert.Data.Case where

import Dagobert.Data.Common (Common', common')
import Record (merge)

type Case =
  { id             :: Int
  , name           :: String
  , classification :: String
  , summary        :: String
  | Common'
  }

type CaseStub =
  { id             :: Int
  , name           :: String
  , classification :: String
  , summary        :: String
  }

newCase :: Case
newCase = merge common' { id: 0, name: "", classification: "", summary: "" }