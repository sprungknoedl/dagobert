module Dagobert.Data.User where

import Dagobert.Data.Common (Common, common)
import Record (merge)

type User =
  { id      :: Int
  , name    :: String
  , company :: String
  , role    :: String
  , email   :: String
  , phone   :: String
  , notes   :: String
  | Common
  }

type UserStub =
  { id      :: Int
  , name    :: String
  , company :: String
  , role    :: String
  , email   :: String
  , phone   :: String
  , notes   :: String
  }

newUser :: User
newUser = merge common { id: 0, name: "", company: "", role: "", email: "", phone: "", notes: "" }