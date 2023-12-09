module Dagobert.Data.Task where

import Dagobert.Data.Common (Common, common)
import Record (merge)

type Task =
  { id        :: Int
  , type      :: String
  , task      :: String
  , done      :: Boolean
  , owner     :: String
  , dateDue   :: String
  | Common
  }

type TaskStub = 
  { id        :: Int
  , type      :: String
  , task      :: String
  , done      :: Boolean
  , owner     :: String
  , dateDue   :: String
  }

newTask :: Task
newTask = merge common { id: 0, type: "", task: "", done: false, owner: "", dateDue: "" }

taskTypes :: Array String
taskTypes = ["Information request", "Analysis", "Deliverable", "Checkpoint", "Other"]