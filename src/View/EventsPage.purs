module Dagobert.View.EventsPage where

import Prelude

import Dagobert.Data.Asset (Asset)
import Dagobert.Data.Event (Event, EventStub, directionValues, eventTypes, newEvent)
import Dagobert.Route (Route(..))
import Dagobert.Utils.Env (Env)
import Dagobert.Utils.Forms (Form, dummyField, form, label, poll, render, selectField, textField, textareaField, validate)
import Dagobert.Utils.HTML (modal, printDateTime)
import Dagobert.Utils.Validation as V
import Dagobert.Utils.XHR as XHR
import Dagobert.View.EntityPage (PageState, DialogControls, entityPage)
import Data.Array ((:))
import Data.Maybe (Maybe(..), maybe)
import Deku.Core (Nut)
import Deku.DOM as D
import Deku.Do as Deku
import Deku.Hooks (useHot, (<#~>))
import Effect (Effect)
import FRP.Poll (Poll)
import Type.Proxy (Proxy(..))

detailsHtml = ( Proxy :: Proxy """
  <div class="p-code-snippet">
    <div class="p-code-snippet__header">
      <h5 class="p-code-snippet__title">Raw</h5>
    </div>

    <pre class="p-code-snippet__block is-wrapped"><code> ~raw~ </code></pre>
  </div>
""")

eventsPage :: { poll ∷ Poll (PageState Event), push ∷ PageState Event -> Effect Unit } -> Env -> Nut
eventsPage state { kase } = Deku.do
  kase <#~> maybe mempty (\c -> entityPage
    { title: ViewTimeline c.id
    , ctor: newEvent
    , id: _.id
    , csv: "/api/case/" <> show c.id <> "/event.csv"
    , fetch:          XHR.get    ("/api/case/" <> show c.id <> "/event")
    , create: \obj -> XHR.post   ("/api/case/" <> show c.id <> "/event") obj
    , update: \obj -> XHR.put    ("/api/case/" <> show c.id <> "/event/" <> show obj.id) obj
    , delete: \obj -> XHR.delete ("/api/case/" <> show c.id <> "/event/" <> show obj.id)
    , hydrate:        XHR.get    ("/api/case/" <> show c.id <> "/asset")

    , columns: [ { title: "Date/Time",     width: "12rem", renderString: _.time >>> printDateTime, renderNut: _.time >>> printDateTime >>> D.text_  }
              , { title: "Type",          width: "10rem", renderString: _.type,                   renderNut: _.type >>> D.text_  }
              , { title: "Event System",  width: "12rem", renderString: _.assetA,                 renderNut: _.assetA >>> D.text_ }
              , { title: "Remote System", width: "12rem", renderString: _.assetB,                 renderNut: \elem -> D.text_ $ elem.direction <> " " <> elem.assetB }
              , { title: "Event",         width: "auto",  renderString: _.event,                  renderNut: _.event >>> D.text_ }
              ]

    , modal: eventModal
    } state)

eventModal :: DialogControls EventStub -> Event -> Array Asset -> Nut
eventModal { save, cancel } input assets = Deku.do
  id        <- useHot input.id
  time      <- useHot input.time
  type_     <- useHot input.type
  assetA    <- useHot input.assetA
  assetB    <- useHot input.assetB
  direction <- useHot input.direction
  event     <- useHot input.event
  raw       <- useHot input.raw

  let
    formBuilder :: Form (Maybe EventStub)
    formBuilder = ado
      id' <- dummyField id
        # validate V.id

      time' <- textField time
        # validate (V.required >=> V.datetime)
        # label "Date / Time"

      type' <- selectField eventTypes type_
        # validate V.required
        # label "Type"

      assetA' <- selectField (map _.name assets) assetA
        # validate V.required
        # label "Event System"

      direction' <- selectField directionValues direction
        # validate V.optional
        # label "Direction"

      assetB' <- selectField ("" : map _.name assets) assetB
        # validate V.optional
        # label "Remote System"

      event' <- textareaField event
        # validate V.required
        # label "Event"

      raw' <- textareaField raw
        # validate V.optional
        # label "Raw"

      in { id: _,  time: _,  type: _,  assetA: _,  assetB: _,  direction: _,  event: _,  raw: _ }
       <$> id' <*> time' <*> type' <*> assetA' <*> assetB' <*> direction' <*> event' <*> raw'

    onSubmit :: Poll (Effect Unit)
    onSubmit = do
      (poll formBuilder) <#> case _ of
        Nothing -> pure unit
        Just obj -> save obj

    onReset :: Poll (Effect Unit)
    onReset =
      pure cancel

    title :: forall r. {id :: Int | r} -> String
    title {id: 0} = "Add event"
    title {id: _} = "Edit event"
  
  modal $ form (title input) (render formBuilder) onSubmit onReset