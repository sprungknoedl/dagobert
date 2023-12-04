module Dagobert.View.CasePage where

import Prelude

import Dagobert.Data.Case (Case, newCase)
import Dagobert.Route (Route(..), routeToTitle)
import Dagobert.Utils.Forms (Form, dummyField, headingField, label, poll, render, textField, textareaField, validate)
import Dagobert.Utils.HTML (css, primaryButton)
import Dagobert.Utils.Hooks ((<~))
import Dagobert.Utils.Validation as V
import Dagobert.Utils.XHR as XHR
import Dagobert.View.EntityPage (PageState(..))
import Data.Array (head)
import Data.Either (Either(..))
import Data.Maybe (Maybe(..), fromMaybe)
import Deku.Core (Nut)
import Deku.DOM as D
import Deku.DOM.Listeners as DL
import Deku.Do as Deku
import Deku.Hooks (useHot, (<#~>))
import Effect (Effect)
import Effect.Aff (Aff, launchAff_)
import FRP.Poll (Poll)

casePage :: { poll ∷ Poll (PageState Case), push ∷ PageState Case -> Effect Unit } -> Nut
casePage state = state.poll <#~> case _ of
  -- ----------------------------------------------------
  Loading ->
  -- ----------------------------------------------------
    D.main [ css "p-4 grow" ] $
      [ D.nav [ css "flex items-center justify-between mb-4" ]
        [ D.h3 [ css "font-bold text-2xl ml-2" ] [ D.text_ $ routeToTitle ViewCaseInfo ] ]
      , D.section [ css "flex flex-col gap-6 max-w-screen-md" ] [ D.text_ "Loading ..." ]
      ]

  -- ----------------------------------------------------
  Error err ->
  -- ----------------------------------------------------
    D.main [ css "p-4 grow" ] $
      [ D.nav [ css "flex items-center justify-between mb-4" ]
        [ D.h3 [ css "font-bold text-2xl ml-2" ] [ D.text_ $ routeToTitle ViewCaseInfo ] ]
      , D.section [ css "flex flex-col gap-6 max-w-screen-md" ] [ D.text_ err ]
      ]

  -- ----------------------------------------------------
  Loaded list -> Deku.do
  -- ----------------------------------------------------
    let case_ = fromMaybe newCase $ head list

    id             <- useHot case_.id
    name           <- useHot case_.name
    classification <- useHot case_.classification
    summary        <- useHot case_.summary

    let
      reload :: Aff Unit
      reload = do
        state <~Loading
        resp <- XHR.get "/api/case"
        case resp of 
          Right list' -> state <~ (Loaded list')
          Left err ->  state <~ (Error err)

      save :: Poll (Effect Unit)
      save = (poll formBuilder) <#> case _ of
        Nothing -> pure unit
        Just obj -> launchAff_ do
          state <~Loading

          resp <- if obj.id == 0
            then XHR.post "/api/case" obj
            else XHR.put ("/api/case/" <> (show obj.id)) obj
          case resp of
            Right _ -> reload
            Left err -> state <~(Error err)

      formBuilder :: Form (Maybe Case)
      formBuilder = ado
        _ <- headingField "General information"
        id' <- dummyField id
          # validate V.id
        name' <- textField name
          # validate V.required
          # label "Case name"
        classification' <- textField classification
          # validate V.optional
          # label "Classification"
        summary' <- textareaField summary
          # validate V.optional
          # label "Summary"

        in { id: _,  name: _,  classification: _,  summary: _ }
        <$> id' <*> name' <*> classification' <*> summary'

    D.main [ css "p-4 grow" ] $
      [ D.nav [ css "flex items-center justify-between mb-4" ]
        [ D.h3 [ css "font-bold text-2xl ml-2" ] [ D.text_ $ routeToTitle ViewCaseInfo ] ]
      , D.section [ css "flex flex-col gap-6 max-w-screen-md" ] [ render formBuilder ]
      , D.footer [ css "mt-8 py-8 border-t border-t-slate-700 max-w-screen-md" ] 
        [ primaryButton [ DL.runOn DL.click save] [ D.text_ "Save" ]
        ]
      ]