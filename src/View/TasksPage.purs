module Dagobert.View.TasksPage where

import Prelude

import Dagobert.Data.Task (Task, TaskStub, newTask, taskTypes)
import Dagobert.Data.User (User)
import Dagobert.Route (Route(..))
import Dagobert.Utils.Env (Env)
import Dagobert.Utils.Forms (Form, checkboxField, dummyField, form, label, poll, render, selectField, textField, textareaField, validate)
import Dagobert.Utils.HTML (css, modal, printDate, renderDateAdded)
import Dagobert.Utils.Icons (checkCircle, clipboardCheck, documentText, magnifyingGlass, questionMarkCircle, xCircle)
import Dagobert.Utils.Validation as V
import Dagobert.Utils.XHR as XHR
import Dagobert.View.EntityPage (DialogControls, PageState, entityPage)
import Data.Array ((:))
import Data.Maybe (Maybe(..), maybe)
import Deku.Core (Nut, fixed)
import Deku.DOM as D
import Deku.Do as Deku
import Deku.Hooks (useHot, (<#~>))
import Effect (Effect)
import FRP.Poll (Poll)

tasksPage :: { poll ∷ Poll (PageState Task), push ∷ PageState Task -> Effect Unit } -> Env -> Nut
tasksPage state { kase } = Deku.do
  let
    renderType :: String -> Nut
    renderType "Information request" = fixed [ questionMarkCircle (css "inline-block w-6 h-6 mr-2"), D.text_ "Information request" ]
    renderType "Analysis"            = fixed [ magnifyingGlass (css "inline-block w-6 h-6 mr-2"), D.text_ "Analysis" ]
    renderType "Deliverable"         = fixed [ documentText (css "inline-block w-6 h-6 mr-2"), D.text_ "Deliverable" ]
    renderType "Checkpoint"          = fixed [ clipboardCheck (css "inline-block w-6 h-6 mr-2"), D.text_ "Checkpoint" ]
    renderType t                     = fixed [ questionMarkCircle (css "inline-block w-6 h-6 mr-2"), D.text_ t ]

    renderDone :: Boolean -> Nut
    renderDone true = checkCircle $ css "w-6 h-6 text-green-500"
    renderDone false = xCircle $ css "w-6 h-6 text-red-500"

  kase <#~> maybe mempty (\c -> entityPage
    { title: ViewTasks c.id
    , ctor: newTask
    , id: _.id
    , csv: "/api/case/" <> show c.id <> "/task.csv"
    , fetch:          XHR.get    ("/api/case/" <> show c.id <> "/task")
    , create: \obj -> XHR.post   ("/api/case/" <> show c.id <> "/task") obj
    , update: \obj -> XHR.put    ("/api/case/" <> show c.id <> "/task/" <> show obj.id) obj
    , delete: \obj -> XHR.delete ("/api/case/" <> show c.id <> "/task/" <> show obj.id)
    , hydrate:        XHR.get    ("/api/case/" <> show c.id <> "/user")

    , columns: [ { title: "Date added", width: "7rem",  renderString: _.dateAdded >>> printDate, renderNut: renderDateAdded  }
               , { title: "Date due",   width: "7rem",  renderString: _.dateDue >>> printDate,   renderNut: _.dateDue >>> printDate >>> D.text_  }
               , { title: "Type",       width: "15rem", renderString: _.type,                    renderNut: _.type >>> renderType }
               , { title: "Task",       width: "auto",  renderString: _.task,                    renderNut: _.task >>> D.text_ }
               , { title: "Owner",      width: "auto",  renderString: _.owner,                   renderNut: _.owner >>> D.text_ }
               , { title: "Done",       width: "7rem",  renderString: _.done >>> show,           renderNut: _.done >>> renderDone }
               ]

    , modal: taskModal
    } state)

taskModal :: DialogControls TaskStub -> Task -> Array User -> Nut
taskModal { save, cancel } input users = Deku.do
  id        <- useHot input.id
  dateDue   <- useHot input.dateDue
  type_     <- useHot input.type
  task      <- useHot input.task
  done      <- useHot input.done
  owner     <- useHot input.owner

  let
    formBuilder :: Form (Maybe TaskStub)
    formBuilder = ado
      id' <- dummyField id
        # validate V.id

      type' <- selectField taskTypes type_
        # validate V.required
        # label "Type"

      task' <- textareaField task
        # validate V.required
        # label "Task"

      owner' <- selectField ("" : map _.name users) owner
        # validate V.optional
        # label "Owner"

      dateDue' <- textField dateDue
        # validate (V.optional >=> V.defaultsTo "1970-01-01T00:00:00Z" >=> V.datetime)
        # label "Date Due"

      done' <- checkboxField done
        # validate V.optional
        # label "Done"

      in { id: _,  dateDue: _,  type: _,  done: _,  owner: _,  task: _ }
       <$> id' <*> dateDue' <*> type' <*> done' <*> owner' <*> task'

    onSubmit :: Poll (Effect Unit)
    onSubmit = do
      (poll formBuilder) <#> case _ of
        Nothing -> pure unit
        Just obj -> save obj

    onReset :: Poll (Effect Unit)
    onReset =
      pure cancel

    title :: forall r. {id :: Int | r} -> String
    title {id: 0} = "Add task"
    title {id: _} = "Edit task"
  
  modal $ form (title input) (render formBuilder) onSubmit onReset