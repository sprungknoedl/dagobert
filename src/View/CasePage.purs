module Dagobert.View.CasePage where

import Prelude

import Dagobert.Data.Case (Case, CaseStub, newCase)
import Dagobert.Route (Route(..))
import Dagobert.Utils.Env (Env)
import Dagobert.Utils.Forms (Form, dummyField, form, label, poll, render, textField, textareaField, validate)
import Dagobert.Utils.HTML (modal)
import Dagobert.Utils.Validation as V
import Dagobert.Utils.XHR as XHR
import Dagobert.View.EntityPage (DialogControls, PageState, entityPage)
import Data.Maybe (Maybe(..))
import Deku.Core (Nut)
import Deku.DOM as D
import Deku.DOM.Attributes as DA
import Deku.DOM.Listeners as DL
import Deku.Do as Deku
import Deku.Hooks (useHot, (<#~>))
import Effect (Effect)
import FRP.Poll (Poll)

casePage :: { poll ∷ Poll (PageState Case), push ∷ PageState Case -> Effect Unit } -> Env -> Nut
casePage state { kase, setKase } = Deku.do
  let
    renderSelect :: Case -> Nut
    renderSelect obj = kase <#~> \cur -> case eq obj <$> cur of
        Just true -> D.span [ DA.klass_ "text-green-500" ] [ D.text_ "▶ Selected" ]
        _         -> D.a [ DA.klass_ "cursor-pointer text-slate-400 hover:text-slate-200 hover:underline", DL.runOn_ DL.click $ setKase (Just obj) ] [ D.text_ "Switch" ]

  entityPage
    { title: ViewCases
    , ctor: newCase
    , id: _.id
    , csv: "/api/case.csv"
    , fetch:          XHR.get    ("/api/case")
    , create: \obj -> XHR.post   ("/api/case") obj
    , update: \obj -> XHR.put    ("/api/case/" <> show obj.id) obj
    , delete: \obj -> XHR.delete ("/api/case/" <> show obj.id)
    , hydrate:        pure $ pure unit

    , columns: [ { title: "",               width: "8rem", renderString: _.id >>> show,    renderNut: renderSelect }
               , { title: "ID",             width: "8rem", renderString: _.id >>> show,    renderNut: _.id >>> show >>> D.text_ }
               , { title: "Name",           width: "auto", renderString: _.name,           renderNut: _.name >>> D.text_  }
               , { title: "Classification", width: "auto", renderString: _.classification, renderNut: _.classification >>> D.text_ }
               , { title: "Summary",        width: "auto", renderString: _.summary,        renderNut: _.summary >>> D.text_ }
               ]

    , modal: caseModal
    } state

caseModal :: DialogControls CaseStub -> Case -> Unit -> Nut
caseModal { save, cancel } input _ = Deku.do
  id             <- useHot input.id
  name           <- useHot input.name
  classification <- useHot input.classification
  summary        <- useHot input.summary

  let
    formBuilder :: Form (Maybe CaseStub)
    formBuilder = ado
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

    onSubmit :: Poll (Effect Unit)
    onSubmit = do
      (poll formBuilder) <#> case _ of
        Nothing -> pure unit
        Just obj -> save obj

    onReset :: Poll (Effect Unit)
    onReset =
      pure cancel

    title :: forall r. {id :: Int | r} -> String
    title {id: 0} = "Add case"
    title {id: _} = "Edit case"
  
  modal $ form (title input) (render formBuilder) onSubmit onReset