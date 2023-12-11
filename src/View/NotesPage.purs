module Dagobert.View.NotesPage where

import Prelude

import Dagobert.Data.Note (Note, NoteStub, newNote)
import Dagobert.Route (Route(..))
import Dagobert.Utils.Env (Env)
import Dagobert.Utils.Forms (Form, dummyField, form, label, poll, render, textField, textareaField, validate)
import Dagobert.Utils.HTML (modal)
import Dagobert.Utils.Validation as V
import Dagobert.Utils.XHR as XHR
import Dagobert.View.EntityPage (DialogControls, PageState, defaultActions, entityPage)
import Data.Maybe (Maybe(..), maybe)
import Deku.Core (Nut)
import Deku.DOM as D
import Deku.Do as Deku
import Deku.Hooks (useHot, (<#~>))
import Effect (Effect)
import FRP.Poll (Poll)

notesPage :: { poll ∷ Poll (PageState Note), push ∷ PageState Note -> Effect Unit } -> Env -> Nut
notesPage state { kase } = Deku.do
  kase <#~> maybe mempty (\c -> entityPage
    { title: ViewNotes c.id
    , ctor: newNote
    , id: _.id
    , csv: "/api/case/" <> show c.id <> "/note.csv"
    , fetch:          XHR.get    ("/api/case/" <> show c.id <> "/note")
    , create: \obj -> XHR.post   ("/api/case/" <> show c.id <> "/note") obj
    , update: \obj -> XHR.put    ("/api/case/" <> show c.id <> "/note/" <> show obj.id) obj
    , delete: \obj -> XHR.delete ("/api/case/" <> show c.id <> "/note/" <> show obj.id)
    , hydrate:        pure $ pure unit

    , columns: [ { title: "Category",    width: "15rem", renderString: _.category,    renderNut: _.category >>> D.text_  }
               , { title: "Title",       width: "15rem", renderString: _.title,       renderNut: _.title >>> D.text_ }
               , { title: "Description", width: "auto",  renderString: _.description, renderNut: _.description >>> D.text_ }
               ]

    , modal: notesModal
    } defaultActions state)

notesModal :: DialogControls NoteStub -> Note -> Unit -> Nut
notesModal { save, cancel } input _ = Deku.do
  id          <- useHot input.id
  category    <- useHot input.category
  title_      <- useHot input.title
  description <- useHot input.description

  let
    formBuilder :: Form (Maybe NoteStub)
    formBuilder = ado
      id' <- dummyField id
        # validate V.id

      category' <- textField category
        # validate V.required
        # label "Category"

      title' <- textField title_
        # validate V.required
        # label "Title"

      description' <- textareaField description
        # validate V.required
        # label "Description"

      in { id: _,  category: _,  title: _,  description: _ }
       <$> id' <*> category' <*> title' <*> description'

    onSubmit :: Poll (Effect Unit)
    onSubmit = do
      (poll formBuilder) <#> case _ of
        Nothing -> pure unit
        Just obj -> save obj

    onReset :: Poll (Effect Unit)
    onReset =
      pure cancel

    title :: forall r. {id :: Int | r} -> String
    title {id: 0} = "Add note"
    title {id: _} = "Edit note"
  
  modal $ form (title input) (render formBuilder) onSubmit onReset