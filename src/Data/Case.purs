module Dagobert.Data.Case where

type Case =
  { id             :: Int
  , name           :: String
  , classification :: String
  , summary        :: String
  }

newCase :: Case
newCase = { id: 0, name: "", classification: "", summary: "" }