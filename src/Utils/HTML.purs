module Dagobert.Utils.HTML where

import Prelude

import Data.Array (index, mapWithIndex, zip)
import Data.Maybe (Maybe(..))
import Data.String (Pattern(..), Replacement(..), codePointFromChar, replace, takeWhile)
import Data.Tuple.Nested ((/\), type (/\))
import Deku.Core (Nut)
import Deku.DOM (Attribute, HTMLInputElement, HTMLButtonElement)
import Deku.DOM as D
import Deku.DOM.Attributes as DA
import Deku.DOM.Listeners as DL
import Deku.Do as Deku
import Deku.Hooks (useState)
import Effect (Effect)
import FRP.Poll (Poll)

css :: ∀ r. String → Poll (Attribute ( klass ∷ String | r ) )
css = DA.klass_

primaryButton :: Array (Poll (Attribute (HTMLButtonElement ()))) -> Array Nut -> Nut
primaryButton attr = D.button $
  [ DA.xtype_ "button", css "px-4 h-10 border border-slate-300 border-transparent rounded-md shadow-sm outline outline-2 outline-offset-2 outline-pink-500 bg-slate-700 hover:bg-pink-500 text-slate-200 text-sm cursor-pointer" ] <> attr

secondaryButton :: Array (Poll (Attribute (HTMLButtonElement ()))) -> Array Nut -> Nut
secondaryButton attr = D.button $
  [ DA.xtype_ "button", css "px-4 h-10 rounded-md shadow-sm  outline outline-2 outline-offset-2 outline-slate-600  bg-slate-700 hover:bg-slate-600 text-slate-200 text-sm cursor-pointer" ] <> attr

dangerButton :: Array (Poll (Attribute (HTMLButtonElement ()))) -> Array Nut -> Nut
dangerButton attr = D.button $
  [ DA.xtype_ "button", css "px-4 h-10 border border-slate-300 border-transparent rounded-md shadow-sm outline outline-2 outline-offset-2 outline-red-700 bg-red-700 hover:bg-red-500 hover:outline-red-500 text-slate-200 text-sm cursor-pointer" ] <> attr

inlineButton :: Array (Poll (Attribute (HTMLButtonElement ()))) -> Array Nut -> Nut
inlineButton attr = D.button $
  [ DA.xtype_ "button", css "p-2 rounded-lg hover:bg-pink-500" ] <> attr

searchInput :: Array (Poll (Attribute (HTMLInputElement ()))) -> Array Nut -> Nut
searchInput attr = D.input $
  [ DA.xtype_ "search", css "px-4 h-10 outline outline-2 outline-offset-2 outline-slate-600 bg-slate-700 text-white rounded-md shadow-sm" ] <> attr

modal :: Nut -> Nut
modal body = D.aside [ css "overflow-y-auto overflow-x-hidden fixed top-0 right-0 z-50 flex justify-center items-center w-full h-full backdrop-blur-lg backdrop-brightness-50" ]
    [ D.div [ css "w-1/2 m-8 bg-slate-800 shadow-xl rounded-xl" ] [ body ] ]

printDate :: String -> String
printDate = takeWhile (not eq $ codePointFromChar 'T')

printDateTime :: String -> String
printDateTime = replace (Pattern "T") (Replacement " ")

tableHead :: Array String -> Array String -> Nut
tableHead columns widths = D.thead_
  [ D.tr_ (mapWithIndex column (zip columns widths)) ]
  where 
    column :: Int -> (String /\ String) -> Nut
    column _ (c /\ w) = D.th [ DA.style_ $ "width: " <> w ] [ D.text_ c ]

sortedTableHead :: forall a b
  . ((a -> b) -> Effect Unit)
  -> Array String 
  -> Array String 
  -> Array (a -> b)
  -> Nut
sortedTableHead cb columns widths fns = Deku.do
  setCol /\ col <- useState (-1)
  let 
    column :: Int -> (String /\ String) -> Nut
    column i (c /\ w) = D.th 
      [ DA.ariaSort $ col <#> (\x -> if x == i then "ascending" else "none")
      , DA.style_ $ "width: " <> w
      , DL.runOn DL.click (onClick i) 
      ] [ D.text_ c ]

    onClick :: Int -> Poll (Effect Unit)
    onClick i = pure (do
      _ <- case index fns i of
        Just fn -> cb fn
        Nothing -> pure unit
      setCol i)

  D.thead_ [ D.tr_ (mapWithIndex column (zip columns widths)) ]

loading :: Nut
loading = mempty

error :: Poll (Effect Unit) -> String -> Nut
error _ _ = mempty