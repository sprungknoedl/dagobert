module Dagobert.View.UsersPage where

import Prelude

import Dagobert.Data.User (User, UserStub, newUser)
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

usersPage :: { poll ∷ Poll (PageState (Array User)), push ∷ PageState (Array User) -> Effect Unit } -> Env -> Nut
usersPage state { kase } = Deku.do
  kase <#~> maybe mempty (\c -> entityPage
    { title: ViewUsers c.id
    , ctor: newUser
    , id: _.id
    , csv:                        "/api/cases/" <> show c.id <> "/users.csv"
    , fetch:          XHR.get    ("/api/cases/" <> show c.id <> "/users")
    , create: \obj -> XHR.post   ("/api/cases/" <> show c.id <> "/users") obj
    , update: \obj -> XHR.put    ("/api/cases/" <> show c.id <> "/users/" <> show obj.id) obj
    , delete: \obj -> XHR.delete ("/api/cases/" <> show c.id <> "/users/" <> show obj.id)
    , hydrate:        pure $ pure unit

    , columns: [ { title: "Name",    width: "auto", renderString: _.name,    renderNut: _.name >>> D.text_ }
               , { title: "Company", width: "auto", renderString: _.company, renderNut: _.company >>> D.text_ }
               , { title: "Role",    width: "auto", renderString: _.role,    renderNut: _.role >>> D.text_ }
               , { title: "Email",   width: "auto", renderString: _.email,   renderNut: _.email >>> D.text_ }
               , { title: "Phone",   width: "auto", renderString: _.phone,   renderNut: _.phone >>> D.text_ }
               , { title: "Notes",   width: "auto", renderString: _.notes,   renderNut: _.notes >>> D.text_ }
               ]

    , modal: userModal
    } defaultActions state)

userModal :: DialogControls UserStub -> User -> Unit -> Nut
userModal { save, cancel } input _ = Deku.do
  id      <- useHot input.id
  name    <- useHot input.name
  company <- useHot input.company
  role    <- useHot input.role
  email   <- useHot input.email
  phone   <- useHot input.phone
  notes   <- useHot input.notes

  let
    formBuilder :: Form (Maybe UserStub)
    formBuilder = ado
      id' <- dummyField id
        # validate V.id

      name' <- textField name
        # validate V.required
        # label "Name"

      company' <- textField company
        # validate V.optional
        # label "Company"

      role' <- textField role
        # validate V.optional
        # label "Role"

      email' <- textField email
        # validate V.optional
        # label "Email"

      phone' <- textField phone
        # validate V.optional
        # label "Phone"

      notes' <- textareaField notes
        # validate V.optional
        # label "Notes"

      in { id: _,  name: _,  company: _,  role: _,  email: _,  phone: _,  notes: _ }
       <$> id' <*> name' <*> company' <*> role' <*> email' <*> phone' <*> notes'

    onSubmit :: Poll (Effect Unit)
    onSubmit = do
      (poll formBuilder) <#> case _ of
        Nothing -> pure unit
        Just obj -> save obj

    onReset :: Poll (Effect Unit)
    onReset =
      pure cancel

    title :: forall r. {id :: Int | r} -> String
    title {id: 0} = "Add user"
    title {id: _} = "Edit user"
  
  modal $ form (title input) (render formBuilder) onSubmit onReset