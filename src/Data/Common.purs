module Dagobert.Data.Common where

import Record (merge)

type Common' =
  ( dateAdded      :: String
  , dateModified   :: String
  , userAdded      :: String
  , userModified   :: String
  )

type Common =
  ( caseId       :: Int
  | Common'
  )

common' :: Record Common'
common' = { dateAdded: "1970-01-01T00:00:00Z", dateModified: "1970-01-01T00:00:00Z", userAdded: "unknown", userModified: "unknown" }

common :: Record Common
common = merge common' { caseId: 0 }