module Dagobert.Data.Note where

type Note =
  { id          :: Int
  , title       :: String
  , category    :: String
  , description :: String
  }

newNote :: Note
newNote = { id: 0, title: "", category: "", description: "" }