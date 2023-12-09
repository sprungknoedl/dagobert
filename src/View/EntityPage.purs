module Dagobert.View.EntityPage where

import Prelude

import Dagobert.Route (Route, routeToTitle)
import Dagobert.Utils.HTML (css, inlineButton, primaryButton, searchInput, secondaryButton)
import Dagobert.Utils.Hooks ((<~))
import Dagobert.Utils.Icons (arrowPath, chevronDown, faceFrown, magnifyingGlass, pencil, plus, trash)
import Dagobert.View.ConfirmDialog (confirmDialog)
import Data.Array (any, filter, index, mapWithIndex, null, sortWith)
import Data.Either (Either(..))
import Data.Maybe (Maybe(..))
import Data.String (Pattern(..), contains)
import Data.Tuple (Tuple(..), uncurry)
import Data.Tuple.Nested ((/\), type (/\))
import Deku.Core (Nut, fixed)
import Deku.DOM as D
import Deku.DOM.Attributes as DA
import Deku.DOM.Listeners as DL
import Deku.Do as Deku
import Deku.Hooks (guard, useHot, useState, useState', (<#~>))
import Effect (Effect)
import Effect.Aff (Aff, launchAff_)
import Effect.Class (liftEffect)
import FRP.Poll (Poll)

data PageState a = Loading
                 | Loaded (Array a)
                 | Error String

type Dialogs a =
  { editDialog   :: a -> Effect Unit
  , deleteDialog :: a -> Effect Unit 
  }

type DialogControls a =
  { save   ∷ a -> Effect Unit
  , cancel ∷ Effect Unit
  }

type Column a =
  { title        :: String
  , width        :: String
  , renderNut    :: a -> Nut
  , renderString :: a -> String
  }

type PageArgs a a' b = 
  { title      :: Route
  , ctor       :: a
  , id         :: a' -> Int
  , fetch      :: Aff (Either String (Array a))
  , create     :: a' -> Aff (Either String a')
  , update     :: a' -> Aff (Either String a')
  , delete     :: a -> Aff (Either String Unit)
  , hydrate    :: Aff (Either String b)

  , modal      :: DialogControls a' -> a -> b -> Nut
  , columns    :: Array (Column a)
  }

entityPage :: forall a a' b. PageArgs a a' b -> { poll ∷ Poll (PageState a), push ∷ PageState a -> Effect Unit } -> Nut
entityPage args pageState = Deku.do
  setModalVisible   /\ modalVisible   <- useState false
  setConfirmVisible /\ confirmVisible <- useState false

  setHydration      /\ hydration      <- useState'

  setSelected       /\ selected       <- useHot args.ctor
  setSearchTerm     /\ searchTerm     <- useState ""
  setSortCol        /\ sortCol        <- useState (-1)

  let
    save :: a' -> Effect Unit
    save obj = launchAff_ do
      pageState <~Loading

      resp1 <- if (eq 0 <<< args.id) obj
        then args.create obj
        else args.update obj
      case resp1 of
        Right _ -> reload
        Left err -> pageState <~(Error err)

    delete :: a -> Effect Unit
    delete obj = launchAff_ do
      pageState <~Loading

      resp1 <- args.delete obj
      case resp1 of
        Right _ -> reload
        Left err -> pageState <~(Error err)

    reload :: Aff Unit
    reload = do
      pageState <~Loading
      resp <- args.fetch
      case resp of 
        Right list -> pageState <~ (Loaded list)
        Left err ->  pageState <~ (Error err)

      liftEffect $ hide

    hide :: Effect Unit
    hide = do
      setModalVisible false
      setConfirmVisible false

    editDialog :: a -> Effect Unit
    editDialog obj = do
      setSelected obj
      setModalVisible true

      launchAff_ do
        d <- args.hydrate
        case d of 
          Right x -> liftEffect $ setHydration x
          Left _  -> pure unit

    deleteDialog :: a -> Effect Unit
    deleteDialog obj = do
      setSelected obj
      setConfirmVisible true

    entityListPanel :: Array Nut -> Nut
    entityListPanel content =
      D.main [css "p-4 grow"] $
        [ D.nav [css "flex items-center justify-between mb-4"]
          [ D.h3 [css "font-bold text-2xl ml-2"] [ D.text_ (routeToTitle args.title) ]
          , D.div [css "flex gap-5 items-center"]
            [ magnifyingGlass (css "w-6 h-6")
            , searchInput [DA.style_ "width: 32rem", DA.placeholder_ "Search", DL.valueOn_ DL.input $ setSearchTerm] []
            , secondaryButton [DL.runOn_ DL.click $ launchAff_ reload] 
              [ arrowPath (css "inline-block mr-1 w-5 h-5")
              , D.text_ "Refresh"
              ]
            , primaryButton [DL.runOn_ DL.click $ editDialog args.ctor] 
              [ plus (css "inline-block mr-1 w-5 h-5")
              , D.text_ "Add"
              ]
            ]
          ]
        ] <> content

    sortedTableHead :: Nut
    sortedTableHead = Deku.do
      let 
        column :: Int -> Column a -> Nut
        column i c = D.th 
          [ css "p-2 text-left cursor-pointer text-slate-400 hover:text-white hover:underline"
          , DA.style_ $ "width: " <> c.width
          , DL.runOn DL.click (onClick i) 
          ] 
          [ D.text_ c.title
          , sortCol <#~> (\x -> if x == i then chevronDown (css "inline-block ml-1 w-4 h-4") else mempty) 
          ]

        onClick :: Int -> Poll (Effect Unit)
        onClick i = pure $ setSortCol i

      D.thead [css "border-b-2 border-b-slate-600"] [ D.tr [css "p-8"] $ (mapWithIndex column args.columns) <> [D.th [DA.style_ $ "width: 7rem"] []]]

    filteringSortPoll :: String /\ Int -> Array a -> Array a
    filteringSortPoll (f /\ s) = 
      (filter (searchFn f) >>> sortWith (sorterFn s))
      where
      sorterFn :: Int -> (a -> String)
      sorterFn col = case index (args.columns <#> _.renderString) col of
        Just fn -> fn
        Nothing -> const ""

      searchFn :: String -> a -> Boolean
      searchFn term a = any identity $ 
        (args.columns <#> _.renderString) <#>
        \fn -> contains (Pattern term) (fn a)

    renderElem :: a -> Nut
    renderElem elem =
      D.tr [css "hover:bg-slate-700"] $
        map (\c -> D.td [css "p-2"] [c.renderNut elem]) args.columns
        <> [ D.td [css "p-2 flex gap-2 justify-end" ]
           [ inlineButton [ DL.runOn_ DL.click $ editDialog elem ] [ pencil $ css "w-4 h-4"]
           , inlineButton [ DL.runOn_ DL.click $ deleteDialog elem ] [ trash $ css "w-4 h-4"]
           ]]

  pageState.poll <#~> case _ of
    -- ----------------------------------------------------
    Loading -> fixed
    -- ----------------------------------------------------
      [ entityListPanel
        [ D.table [css "table-auto w-full" ] 
          [ sortedTableHead
          , D.caption [ css "caption-bottom w-1/3 my-4 mx-auto" ] 
            [ D.h3 [ css "m-2 text-xl text-slate-400" ] 
              [ arrowPath $ css "inline-block w-6 h-6 mr-2"
              , D.text_ "Loading ..."
              ] 
            , D.p_ [ D.text_ "We're getting the page in shape, hang in there." ]
            ]            
          ]
        ]
      , guard modalVisible   $ (Tuple <$> selected <*> hydration) <#~> (uncurry $ args.modal { save: save, cancel: hide })
      ]

    -- ----------------------------------------------------
    Loaded list -> fixed
    -- ----------------------------------------------------
      [ entityListPanel
        [ D.table [css "table-auto w-full" ] 
          [ sortedTableHead
          , (Tuple <$> searchTerm <*> sortCol) <#~> \fs -> D.tbody_ $ (filteringSortPoll fs) list <#> renderElem
          , if null list 
            then D.caption [ css "caption-bottom w-1/3 my-4 mx-auto" ] 
              [ D.h3 [ css "mb-2 mt-4 text-xl text-slate-400" ] 
                [ faceFrown $ css "inline-block w-6 h-6 mr-2"
                , D.text_ "Nothing here ..."
                ] 
              , D.p [ css "mb-4" ] [ D.text_ "It looks empty here. Try adding elements to this page ↓" ]
              , primaryButton [ DL.runOn_ DL.click $ editDialog args.ctor ] 
                [ plus (css "inline-block mr-1 w-5 h-5")
                , D.text_ "Add"
                ]
              ] 
            else mempty
          ]
        ]
      , guard modalVisible   $ (Tuple <$> selected <*> hydration) <#~> (uncurry $ args.modal { save: save, cancel: hide })
      , guard confirmVisible $ selected                           <#~> confirmDialog { accept: delete, reject: hide }
      ]

    -- ----------------------------------------------------
    Error err -> fixed
    -- ----------------------------------------------------
      [ entityListPanel
        [ D.table [css "table-auto w-full" ] 
          [ sortedTableHead
          , D.caption [ css "caption-bottom w-1/3 my-4 mx-auto" ] 
            [ D.h3 [ css "mb-2 mt-4 text-xl text-red-500" ] 
              [ faceFrown $ css "inline-block w-6 h-6 mr-2"
              , D.text_ "Oops ..."
              ] 
            , D.p [ css "mb-4" ] [ D.text_ "I'm sorry, but there seems to be an critical error:" ]
            , D.pre [ css "mb-4 p-4 bg-slate-900 rounded-md" ] [ D.text_ err ]
            , secondaryButton [ DL.runOn_ DL.click $ (launchAff_ reload) ] 
              [ arrowPath (css "inline-block mr-1 w-5 h-5")
              , D.text_ "Reload"
              ]
            ] 
          ]
        ]
      , guard modalVisible   $ (Tuple <$> selected <*> hydration) <#~> (uncurry $ args.modal { save: save, cancel: hide })
      , guard confirmVisible $ selected                           <#~> confirmDialog { accept: delete, reject: hide }
      ]