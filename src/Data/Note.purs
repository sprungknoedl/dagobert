module Dagobert.Data.Note where

import Dagobert.Data.Common (Common, common)
import Record (merge)

type Note =
  { id          :: Int
  , title       :: String
  , category    :: String
  , description :: String
  | Common
  }

type NoteStub =
  { id          :: Int
  , title       :: String
  , category    :: String
  , description :: String
  }

newNote :: Note
newNote = merge common { id: 0, title: "", category: "", description: "" }