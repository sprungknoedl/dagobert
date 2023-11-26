module Dagobert.Data.Task where

type Task =
  { id        :: Int
  , type      :: String
  , task      :: String
  , done      :: Boolean
  , owner     :: String
  , dateAdded :: String
  , dateDue   :: String
  }

newTask :: Task
newTask = { id: 0, type: "", task: "", done: false, owner: "", dateAdded: "", dateDue: "" }

taskTypes :: Array String
taskTypes = ["Information request", "Analysis", "Deliverable", "Checkpoint", "Other"]