module Dagobert.Utils.Forms where

import Prelude

import Dagobert.Utils.HTML (primaryButton, secondaryButton)
import Dagobert.Utils.Icons (xMark)
import Dagobert.Utils.Validation (Validator)
import Data.Array ((:))
import Data.Either (Either(..), hush)
import Data.Filterable (filter)
import Data.Foldable (for_)
import Data.Maybe (Maybe)
import Data.Tuple.Nested (type (/\), (/\))
import Deku.Core (Nut)
import Deku.DOM (Attribute)
import Deku.DOM as D
import Deku.DOM.Attributes as DA
import Deku.DOM.Listeners as DL
import Deku.Hooks ((<#~>))
import Effect (Effect)
import FRP.Poll (Poll)
import Web.Event.Event as Web
import Web.HTML.HTMLSelectElement as SelectElement
import Web.HTML.HTMLTextAreaElement as TextareaElement

form ∷ String -> Nut -> Poll (Effect Unit) -> Poll (Effect Unit) -> Nut
form title controls onSubmit onReset =
  D.form_
    [ D.header [ DA.klass_ "p-6 px-8 border-b border-b-slate-700 flex justify-between" ]
      [ D.h4  [ DA.klass_ "font-bold text-slate-200" ] [ D.text_ title ]
      , D.div_
          [ D.button [ DA.xtype_ "button", DL.runOn DL.click onReset ]
          [ xMark $ DA.klass_ "w-6 h-6" ] 
          ]
      ]
      
    , D.section [ DA.klass_ "p-8 flex flex-col gap-6" ] [ controls ]

    , D.footer [ DA.klass_ "p-6 px-8 border-t border-t-slate-700 flex gap-4" ] 
      [ primaryButton [ DA.xtype_ "submit", DL.runOn DL.click onSubmit ] [ D.text_ "Save" ]
      , secondaryButton [ DA.xtype_ "button", DL.runOn DL.click onReset ] [ D.text_ "Cancel" ]
      ]
    ] 

-- | Runs an effect with the `value` property of the target element when it triggers the given event.
valueOnInput :: forall r
   . ( Poll ( Web.Event -> Effect Unit ) -> Poll ( Attribute r ) )
  -> Poll ( String -> Effect Unit )
  -> Poll ( Attribute r )
valueOnInput = DL.valueOn

-- | Runs an effect with the `value` property of the target element when it triggers the given event.
valueOnSelect :: forall r
   . ( Poll ( Web.Event -> Effect Unit ) -> Poll ( Attribute r ) )
  -> Poll ( String -> Effect Unit )
  -> Poll ( Attribute r )
valueOnSelect listener =
  listener <<< map \push e -> for_ ( Web.target e >>= SelectElement.fromEventTarget ) $ SelectElement.value >=> push

-- | Runs an effect with the `value` property of the target element when it triggers the given event.
valueOnTextarea :: forall r
   . ( Poll ( Web.Event -> Effect Unit ) -> Poll ( Attribute r ) )
  -> Poll ( String -> Effect Unit )
  -> Poll ( Attribute r )
valueOnTextarea listener =
  listener <<< map \push e -> for_ ( Web.target e >>= TextareaElement.fromEventTarget ) $ TextareaElement.value >=> push

newtype Form a = Form
  { ui :: Nut
  , poll :: Poll a
  }

derive instance functorForm :: Functor Form

instance applyForm :: Apply Form where
  apply (Form a) (Form b) = Form
    { ui: a.ui <> b.ui
    , poll: a.poll <*> b.poll
    }

instance applicativeForm :: Applicative Form where
  pure a = Form 
    { ui: mempty
    , poll: pure a
    }

render :: forall a. Form a -> Nut
render (Form f) = f.ui

poll :: forall a. Form a -> Poll a
poll (Form f) = f.poll

dummyField :: forall a. (a -> Effect Unit) /\ (Poll a) -> Form a
dummyField (_ /\ value) = Form
  { ui: mempty
  , poll: value
  }

headingField :: String -> Form Unit
headingField title = Form
  { ui: D.h4 [ DA.klass_ "font-bold text-lg border-b border-b-slate-600 my-4" ] [ D.text_ title ]
  , poll: pure unit
  }

subtitle :: forall a. String -> Form a -> Form a
subtitle desc (Form f) = Form
  { ui: f.ui <> D.p [ DA.klass_ "-mt-8 mb-4" ] [ D.text_ desc ]
  , poll: f.poll
  }

help :: forall a. String -> Form a -> Form a
help desc (Form f) = Form
  { ui: f.ui <> D.p [ DA.klass_ "text-slate-400 mt-2" ] [ D.text_ desc ]
  , poll: f.poll
  }

textField :: (String -> Effect Unit) /\ (Poll String) -> Form String
textField (pusher /\ value) = Form
  { ui: D.input
    [ DA.xtype_ "text"
    , DA.klass_ "px-4 w-full h-10 outline outline-2 outline-offset-2 outline-slate-600 focus:outline-pink-500 bg-slate-700 text-white caret-pink-500 rounded-md shadow-sm"
    , DA.value value
    , valueOnInput DL.input $ pure pusher
    ] []
  , poll: value
  }

dateField :: (String -> Effect Unit) /\ (Poll String) -> Form String
dateField (pusher /\ value) = Form
  { ui: D.input
    [ DA.xtype_ "date"
    , DA.klass_ "px-4 w-full h-10 outline outline-2 outline-offset-2 outline-slate-600 focus:outline-pink-500 bg-slate-700 text-white caret-pink-500 rounded-md shadow-sm"
    , DA.value value
    , valueOnInput DL.input $ pure pusher
    ] []
  , poll: value
  }

datetimeField :: (String -> Effect Unit) /\ (Poll String) -> Form String
datetimeField (pusher /\ value) = Form
  { ui: D.input
    [ DA.xtype_ "datetime"
    , DA.klass_ "px-4 w-full h-10 outline outline-2 outline-offset-2 outline-slate-600 focus:outline-pink-500 bg-slate-700 text-white caret-pink-500 rounded-md shadow-sm"
    , DA.value value
    , valueOnInput DL.input $ pure pusher
    ] []
  , poll: value
  }

checkboxField :: (Boolean -> Effect Unit) /\ (Poll Boolean) -> Form Boolean
checkboxField (pusher /\ value) = Form
  { ui: D.input
    [ DA.xtype_ "checkbox"
    , DA.klass_ "accent-pink-500"
    , DA.checked $ filter identity value $> "true"
    , DL.checkedOn DL.input $ pure pusher
    ] []
  , poll: value
  }

textareaField :: (String -> Effect Unit) /\ (Poll String) -> Form String
textareaField (pusher /\ value) = Form
  { ui: D.textarea
    [ DA.klass_ "p-4 w-full outline outline-2 outline-offset-2 outline-slate-600 focus:outline-pink-500 bg-slate-700 text-white caret-pink-500 rounded-md shadow-sm"
    , valueOnTextarea DL.input $ pure pusher
    , DA.rows_ "5"
    ] [ D.text value ]
  , poll: value
  }

selectField :: Array String -> (String -> Effect Unit) /\ (Poll String) -> Form String
selectField options (pusher /\ value) = Form
  { ui: ( D.select
    [ DA.klass_ "px-4 w-full h-10 outline outline-2 outline-offset-2 outline-slate-600 focus:outline-pink-500 bg-slate-700 text-white rounded-md shadow-sm"
    , valueOnSelect DL.input $ pure pusher
    ] ( renderDefaultOption : map renderOption options ) )
  , poll: value
  }
  where
    renderDefaultOption :: Nut
    renderDefaultOption = D.option
        [ DA.disabled_ "true"
        , DA.selected $ filter (eq "") value $> "true"
        ] [ D.text_ "Select an option" ]

    renderOption :: String -> Nut
    renderOption opt = D.option
        [ DA.value_ opt
        , DA.selected $ filter (eq opt) value $> "true"
        ] [ D.text_ opt ]

label :: forall a. String -> Form a -> Form a
label str (Form f) = Form
  { ui: D.div [ DA.klass_ "flex flex-row" ]
    [ D.div [ DA.klass_ "grow-0 w-56 pt-2" ] [ D.label_ [ D.text_ str ] ]
    , D.div [ DA.klass_ "grow" ] [ f.ui ]
    ]
  , poll: f.poll
  }

validate :: forall a. Validator a -> Form a -> Form (Maybe a)
validate validator (Form f) = Form 
  { ui: f.ui <> ( f.poll <#~> \x -> case validator x of
      Left err -> D.p [ DA.klass_ "text-pink-500 mt-2" ] [ D.text_ err]
      Right _   -> mempty
      )
  , poll: hush <<< validator <$> f.poll
  }