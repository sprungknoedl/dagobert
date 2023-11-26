module Dagobert.View.IndicatorsPage where

import Prelude

import Dagobert.Data.Indicator (Indicator, indicatorTypes, newIndicator, tlpValues)
import Dagobert.Route (Route(..))
import Dagobert.Utils.Forms (Form, dummyField, form, label, poll, render, selectField, textField, textareaField, validate)
import Dagobert.Utils.HTML (css, modal)
import Dagobert.Utils.Icons (commandLine, fingerprint, folderOpen, globeEurope, link, mapPin, questionMarkCircle)
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

indicatorsPage :: { poll ∷ Poll (PageState Indicator), push ∷ PageState Indicator -> Effect Unit } -> Nut
indicatorsPage = Deku.do
  let
    renderType :: String -> Nut
    renderType "IP"      = fixed [ mapPin (css "inline-block w-6 h-6 mr-2"), D.text_ "IP" ]
    renderType "Domain"  = fixed [ globeEurope (css "inline-block w-6 h-6 mr-2"), D.text_ "Domain" ]
    renderType "URL"     = fixed [ link (css "inline-block w-6 h-6 mr-2"), D.text_ "URL" ]
    renderType "Path"    = fixed [ folderOpen (css "inline-block w-6 h-6 mr-2"), D.text_ "Path" ]
    renderType "Hash"    = fixed [ fingerprint (css "inline-block w-6 h-6 mr-2"), D.text_ "Hash" ]
    renderType "Service" = fixed [ commandLine (css "inline-block w-6 h-6 mr-2"), D.text_ "Service" ]
    renderType t         = fixed [ questionMarkCircle (css "inline-block w-6 h-6 mr-2"), D.text_ t ]

  entityPage
    { title: ViewIndicators
    , ctor: newIndicator
    , id: _.id
    , fetch:          XHR.get "/api/indicator"
    , create: \obj -> XHR.post "/api/indicator" obj
    , update: \obj -> XHR.put ("/api/indicator/" <> show obj.id) obj
    , delete: \obj -> XHR.delete ("/api/indicator/" <> show obj.id)
    , hydrate:        pure $ pure unit

    , columns: [ { title: "Date added",  width: "7rem",  renderString: const "1970-01-01", renderNut: const "1970-01-01" >>> D.text_  }
               , { title: "Type",        width: "10rem", renderString: _.type,             renderNut: _.type >>> renderType  }
               , { title: "Value",       width: "auto",  renderString: _.value,            renderNut: _.value >>> D.text_ }
               , { title: "Description", width: "auto",  renderString: _.description,      renderNut: _.description >>> D.text_ }
               , { title: "TLP",         width: "8rem",  renderString: _.tlp,              renderNut: _.tlp >>> D.text_ }
               , { title: "Source",     width: "auto",   renderString: _.source,           renderNut: _.source >>> D.text_ }
               ]

    , modal: indicatorModal
    }

indicatorModal :: DialogControls Indicator -> Indicator -> Unit -> Nut
indicatorModal { save, cancel } input _ = Deku.do
  id          <- useHot input.id
  type_       <- useHot input.type
  value       <- useHot input.value
  description <- useHot input.description
  tlp         <- useHot input.tlp
  source      <- useHot input.source

  let
    formBuilder :: Form (Maybe Indicator)
    formBuilder = ado
      id' <- dummyField id
        # validate V.id

      type' <- selectField indicatorTypes type_
        # validate V.required
        # label "Type"

      value' <- textField value
        # validate V.required
        # label "Value"

      description' <- textareaField description
        # validate V.optional
        # label "Description"

      tlp' <- selectField tlpValues tlp
        # validate V.required
        # label "TLP"

      source' <- textField source
        # validate V.optional
        # label "Source"

      in { id: _,  type: _,  value: _,  tlp: _,  description: _,  source: _}
       <$> id' <*> type' <*> value' <*> tlp' <*> description' <*> source'

    onSubmit :: Poll (Effect Unit)
    onSubmit = do
      (poll formBuilder) <#> case _ of
        Nothing -> pure unit
        Just obj -> save obj

    onReset :: Poll (Effect Unit)
    onReset =
      pure cancel

    title :: forall r. {id :: Int | r} -> String
    title {id: 0} = "Add indicator"
    title {id: _} = "Edit indicator"
  
  modal $ form (title input) (render formBuilder) onSubmit onReset