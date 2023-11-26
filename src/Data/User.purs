module Dagobert.Data.User where

type User =
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
newUser = { id: 0, shortName: "", fullName: "", company: "", role: "", email: "", phone: "", notes: "" }