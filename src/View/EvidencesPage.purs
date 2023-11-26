module Dagobert.View.EvidencesPage where

import Prelude

import Dagobert.Data.Evidence (Evidence, evidenceTypes, newEvidence)
import Dagobert.Route (Route(..))
import Dagobert.Utils.Forms (Form, dummyField, form, label, poll, render, selectField, textField, textareaField, validate)
import Dagobert.Utils.HTML (css, modal)
import Dagobert.Utils.Icons (archivBox, cpuChip, documentText, folderOpen, questionMarkCircle, server)
import Dagobert.Utils.Validation as V
import Dagobert.Utils.XHR as XHR
import Dagobert.View.EntityPage (PageState, DialogControls, entityPage)
import Data.Maybe (Maybe(..))
import Deku.Core (Nut, fixed)
import Deku.DOM as D
import Deku.Do as Deku
import Deku.Hooks (useHot)
import Effect (Effect)
import FRP.Poll (Poll)

evidencesPage :: { poll ∷ Poll (PageState Evidence), push ∷ PageState Evidence -> Effect Unit } -> Nut
evidencesPage = Deku.do
  let
    renderType :: String -> Nut
    renderType "File"                 = fixed [ folderOpen (css "inline-block w-6 h-6 mr-2"), D.text_ "File" ]
    renderType "Log"                  = fixed [ documentText (css "inline-block w-6 h-6 mr-2"), D.text_ "Log" ]
    renderType "Artifacts Collection" = fixed [ archivBox (css "inline-block w-6 h-6 mr-2"), D.text_ "Artifacts Collection" ]
    renderType "System Image"         = fixed [ server (css "inline-block w-6 h-6 mr-2"), D.text_ "System Image" ]
    renderType "Memory Dump"          = fixed [ cpuChip (css "inline-block w-6 h-6 mr-2"), D.text_ "Memory Dump" ]
    renderType t                      = fixed [ questionMarkCircle (css "inline-block w-6 h-6 mr-2"), D.text_ t ]

  entityPage
    { title: ViewEvidences
    , ctor: newEvidence
    , id: _.id
    , fetch:          XHR.get "/api/evidence"
    , create: \obj -> XHR.post "/api/evidence" obj
    , update: \obj -> XHR.put ("/api/evidence/" <> show obj.id) obj
    , delete: \obj -> XHR.delete ("/api/evidence/" <> show obj.id)
    , hydrate:        pure $ pure unit

    , columns: [ { title: "Date added",  width: "7rem",  renderString: const "1970-01-01", renderNut: const "1970-01-01" >>> D.text_  }
               , { title: "Type",        width: "15rem", renderString: _.type,             renderNut: _.type >>> renderType }
               , { title: "Name",        width: "auto", renderString: _.name,             renderNut: _.name >>> D.text_ }
               , { title: "Description", width: "auto",  renderString: _.description,      renderNut: _.description >>> D.text_ }
               , { title: "Hash",        width: "auto",  renderString: _.hash,             renderNut: _.hash >>> D.text_ }
               , { title: "Location",    width: "auto",  renderString: _.location,         renderNut: _.location >>> D.text_ }
               ]

    , modal: evidenceModal
    }

evidenceModal :: DialogControls Evidence -> Evidence -> Unit -> Nut
evidenceModal { save, cancel } input _ = Deku.do
  id          <- useHot input.id
  type_       <- useHot input.type
  name        <- useHot input.name
  description <- useHot input.description
  hash        <- useHot input.hash
  location    <- useHot input.location

  let
    formBuilder :: Form (Maybe Evidence)
    formBuilder = ado
      id' <- dummyField id
        # validate V.id

      type' <- selectField evidenceTypes type_
        # validate V.required
        # label "Type"

      name' <- textField name
        # validate V.required
        # label "Name"

      description' <- textareaField description
        # validate V.optional
        # label "Description"

      hash' <- textField hash
        # validate V.optional
        # label "Hash"

      location' <- textField location
        # validate V.optional
        # label "Location"

      in { id: _,  type: _,  name: _,  description: _,  hash: _,  location: _ }
       <$> id' <*> type' <*> name' <*> description' <*> hash' <*> location'

    onSubmit :: Poll (Effect Unit)
    onSubmit = do
      (poll formBuilder) <#> case _ of
        Nothing -> pure unit
        Just obj -> save obj

    onReset :: Poll (Effect Unit)
    onReset =
      pure cancel
  
    title :: forall r. {id :: Int | r} -> String
    title {id: 0} = "Add evidence"
    title {id: _} = "Edit evidence"

  modal $ form (title input) (render formBuilder) onSubmit onReset