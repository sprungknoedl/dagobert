module Dagobert.Data.User where

import Dagobert.Data.Common (Common, common)
import Record (merge)

type User =
  { id        :: Int
  , shortName :: String
  , fullName  :: String
  , company   :: String
  , role      :: String
  , email     :: String
  , phone     :: String
  , notes     :: String
  | Common
  }

type UserStub =
  { id        :: Int
  , shortName :: String
  , fullName  :: String
  , company   :: String
  , role      :: String
  , email     :: String
  , phone     :: String
  , notes     :: String
  }

newUser :: User
newUser = merge common { id: 0, shortName: "", fullName: "", company: "", role: "", email: "", phone: "", notes: "" }