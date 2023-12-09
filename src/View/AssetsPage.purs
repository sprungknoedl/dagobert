module Dagobert.View.AssetsPage where

import Prelude

import Dagobert.Data.Asset (Asset, assetTypes, compromiseStates, newAsset)
import Dagobert.Route (Route(..))
import Dagobert.Utils.Env (Env)
import Dagobert.Utils.Forms (Form, checkboxField, dummyField, form, label, poll, render, selectField, textField, textareaField, validate)
import Dagobert.Utils.HTML (css, modal)
import Dagobert.Utils.Icons (bug, checkCircle, desktop, questionMarkCircle, server, user, xCircle)
import Dagobert.Utils.Validation as V
import Dagobert.Utils.XHR as XHR
import Dagobert.View.EntityPage (DialogControls, PageState, entityPage)
import Data.Maybe (Maybe(..), maybe)
import Deku.Core (Nut, fixed)
import Deku.DOM as D
import Deku.Do as Deku
import Deku.Hooks (useHot, (<#~>))
import Effect (Effect)
import FRP.Poll (Poll)

assetsPage :: { poll ∷ Poll (PageState Asset) , push ∷ (PageState Asset) -> Effect Unit } -> Env -> Nut
assetsPage state { kase } = Deku.do
  let
    renderType :: String -> Nut
    renderType "Account" = fixed [ user (css "inline-block w-6 h-6 mr-2"), D.text_ "Account" ]
    renderType "Desktop" = fixed [ desktop (css "inline-block w-6 h-6 mr-2"), D.text_ "Desktop" ]
    renderType "Server"  = fixed [ server (css "inline-block w-6 h-6 mr-2"), D.text_ "Server" ]
    renderType t         = fixed [ questionMarkCircle (css "inline-block w-6 h-6 mr-2"), D.text_ t ]

    renderCompromised :: String -> Nut
    renderCompromised value
      | value == "Compromised" = D.span [ css "text-red-500" ] [ bug (css "inline-block w-6 h-6 mr-2"), D.text_ "Yes" ]
      | value == "Not compromised" = D.text_ "No"
      | value == "Unknown" = D.text_ "Unknown"
      | otherwise = D.text_ value

    renderAnalysed :: Boolean -> Nut
    renderAnalysed true = checkCircle $ css "w-6 h-6 text-green-500"
    renderAnalysed false = xCircle $ css "w-6 h-6 text-red-500"

  kase <#~> maybe mempty (\c -> entityPage
    { title: ViewAssets 0
    , ctor: newAsset
    , id: _.id
    , fetch:          XHR.get    ("/api/case/" <> show c.id <> "/asset")
    , create: \obj -> XHR.post   ("/api/case/" <> show c.id <> "/asset") obj
    , update: \obj -> XHR.put    ("/api/case/" <> show c.id <> "/asset/" <> show obj.id) obj
    , delete: \obj -> XHR.delete ("/api/case/" <> show c.id <> "/asset/" <> show obj.id)
    , hydrate:        pure $ pure unit

    , columns: [ { title: "Date added",  width: "7rem",  renderString: const "1970-01-01",  renderNut: const "1970-01-01" >>> D.text_  }
              , { title: "Type",        width: "10rem",  renderString: _.type,              renderNut: _.type >>> renderType  }
              , { title: "Name",        width: "auto",  renderString: _.name,              renderNut: _.name >>> D.text_ }
              , { title: "IP",          width: "10rem", renderString: _.ip,                renderNut: _.ip >>> D.text_ }
              , { title: "Description", width: "auto",  renderString: _.description,       renderNut: _.description >>> D.text_ }
              , { title: "Compromised", width: "8rem",  renderString: _.compromised,       renderNut: _.compromised >>> renderCompromised }
              , { title: "Analysed",    width: "7rem",  renderString: _.analysed >>> show, renderNut: _.analysed >>> renderAnalysed }
              ]

    , modal: assetModal
    } state)

assetModal :: DialogControls Asset -> Asset -> Unit -> Nut
assetModal { save, cancel } input _ = Deku.do
  id          <- useHot input.id
  type_       <- useHot input.type
  name        <- useHot input.name
  ip          <- useHot input.ip
  description <- useHot input.description
  compromised <- useHot input.compromised
  analysed    <- useHot input.analysed

  let
    formBuilder :: Form (Maybe Asset)
    formBuilder = ado
      id' <- dummyField id
        # validate V.id

      type' <- selectField assetTypes type_
        # validate V.required
        # label "Type"

      name' <- textField name
        # validate V.required
        # label "Name"

      ip' <- textField ip
        # validate V.optional
        # label "IP"

      description' <- textareaField description
        # validate V.optional
        # label "Description"

      compromised' <- selectField compromiseStates compromised
        # validate V.required
        # label "Compromised"

      analysed' <- checkboxField analysed
        # validate V.optional
        # label "Aanalysed"

      in { id: _,  type: _,  name: _,  ip: _,  description: _,  compromised: _,  analysed: _ }
       <$> id' <*> type' <*> name' <*> ip' <*> description' <*> compromised' <*> analysed'

    onSubmit :: Poll (Effect Unit)
    onSubmit = do
      (poll formBuilder) <#> case _ of
        Nothing -> pure unit
        Just obj -> save obj

    onReset :: Poll (Effect Unit)
    onReset =
      pure cancel

    title :: forall r. {id :: Int | r} -> String
    title {id: 0} = "Add asset"
    title {id: _} = "Edit asset"
  
  modal $ form (title input) (render formBuilder) onSubmit onReset