module Dagobert.View.EntityPage where

import Prelude

import Dagobert.Route (Route, routeToTitle)
import Dagobert.Utils.HTML (css, inlineButton, primaryButton, searchInput, secondaryButton, secondaryLink)
import Dagobert.Utils.Icons (arrowDownTray, arrowPath, chevronDown, faceFrown, magnifyingGlass, pencil, plus, trash)
import Dagobert.View.ConfirmDialog (confirmDialog)
import Data.Array (any, filter, index, mapWithIndex, null, sortWith)
import Data.Either (Either(..), either)
import Data.Maybe (Maybe(..))
import Data.String (Pattern(..), contains)
import Data.Tuple (Tuple(..))
import Data.Tuple.Nested ((/\), type (/\))
import Deku.Core (Nut, fixed)
import Deku.DOM as D
import Deku.DOM.Attributes as DA
import Deku.DOM.Listeners as DL
import Deku.Do as Deku
import Deku.Hooks (useState, useState', (<#~>))
import Effect (Effect)
import Effect.Aff (Aff, launchAff_)
import Effect.Class (liftEffect)
import FRP.Poll (Poll)

type PollIO a = { poll ∷ Poll a, push ∷ a -> Effect Unit }

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

type Ctx a =
  { setDialog :: Nut -> Effect Unit
  , setState  :: PageState a -> Effect Unit
  , reload    :: Aff Unit
  }

type Column a =
  { title        :: String
  , width        :: String
  , renderNut    :: a -> Nut
  , renderString :: a -> String
  }

type Action a a' b = PageArgs a a' b -> Ctx a -> Nut

type PageArgs a a' b = 
  { title      :: Route
  , ctor       :: a
  , id         :: a' -> Int
  , csv        :: String
  , fetch      :: Aff (Either String (Array a))
  , create     :: a' -> Aff (Either String a')
  , update     :: a' -> Aff (Either String a')
  , delete     :: a -> Aff (Either String Unit)
  , hydrate    :: Aff (Either String b)

  , modal      :: DialogControls a' -> a -> b -> Nut
  , columns    :: Array (Column a)
  }

defaultActions :: forall a a' b. Array (Action a a' b)
defaultActions =
  [ exportCsvAction
  , reloadAction
  , addAction
  ]

addAction :: forall a a' b. PageArgs a a' b -> Ctx a -> Nut
addAction args ctx = Deku.do
  setHydration /\ hydration <- useState'

  let
    save :: a' -> Effect Unit
    save obj = do
      ctx.setState $ Loading
      launchAff_   $ args.create obj >>= either 
        (\err -> liftEffect $ ctx.setState (Error err))
        (const ctx.reload)

    addDialog :: Effect Unit
    addDialog = do
      ctx.setDialog $ hydration <#~> args.modal { save: save, cancel: ctx.setDialog mempty } args.ctor
      launchAff_    $ args.hydrate >>= either (const $ pure unit) (liftEffect <<< setHydration)

  primaryButton [DL.runOn_ DL.click $ addDialog] 
    [ plus (css "inline-block mr-1 w-5 h-5")
    , D.text_ "Add"
    ]

reloadAction :: forall a a' b. PageArgs a a' b -> Ctx a -> Nut
reloadAction _ ctx = Deku.do
  secondaryButton [DL.runOn_ DL.click $ launchAff_ ctx.reload] 
    [ arrowPath (css "inline-block mr-1 w-5 h-5")
    , D.text_ "Refresh"
    ]

exportCsvAction :: forall a a' b. PageArgs a a' b -> Ctx a -> Nut
exportCsvAction args _ =
  secondaryLink [ DA.href_ args.csv ]
    [ arrowDownTray (css "inline-block mr-1 w-5 h-5")
    , D.text_ "Export CSV"
    ]

entityPage :: forall a a' b. PageArgs a a' b -> Array (Action a a' b) -> PollIO (PageState a) -> Nut
entityPage args actions state = Deku.do
  setDialog     /\ dialog     <- useState'
  setHydration  /\ hydration  <- useState'
  setSearchTerm /\ searchTerm <- useState ""
  setSortCol    /\ sortCol    <- useState (-1)

  let
    save :: a' -> Effect Unit
    save obj = do
      state.push $ Loading
      launchAff_ $ args.update obj >>= either 
        (\err -> liftEffect $ state.push (Error err))
        (const reload)

    delete :: a -> Effect Unit
    delete obj = do
      state.push $ Loading
      launchAff_ $ args.delete obj >>= either 
        (\err -> liftEffect $ state.push (Error err))
        (const reload)

    reload :: Aff Unit
    reload = do
      liftEffect $ state.push Loading
      resp <- args.fetch
      case resp of 
        Right list -> liftEffect $ state.push  (Loaded list)
        Left err   -> liftEffect $ state.push  (Error err)

      liftEffect $ setDialog mempty

    editDialog :: a -> Effect Unit
    editDialog obj = do
      setDialog  $ hydration <#~> args.modal { save: save, cancel: setDialog mempty } obj
      launchAff_ $ args.hydrate >>= either (const $ pure unit) (liftEffect <<< setHydration)

    deleteDialog :: a -> Effect Unit
    deleteDialog obj = do
      setDialog $ confirmDialog { accept: delete, reject: setDialog mempty } obj

    entityListPanel :: Array Nut -> Nut
    entityListPanel content =
      D.main [css "p-4 grow"] $
        [ D.nav [css "flex items-center justify-between mb-4"]
          [ D.h3 [css "font-bold text-2xl ml-2"] [ D.text_ (routeToTitle args.title) ]
          , D.div [css "flex gap-5 items-center"]
            ([ magnifyingGlass (css "w-6 h-6")
            , searchInput [DA.style_ "width: 32rem", DA.placeholder_ "Search", DL.valueOn_ DL.input $ setSearchTerm] []
            ] <> map (\a -> a args ctx) actions)
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

    ctx :: Ctx a
    ctx = { setDialog: setDialog, setState: state.push, reload: reload }

  state.poll <#~> case _ of
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
      , dialog <#~> identity
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
              , addAction args ctx
              ] 
            else mempty
          ]
        ]
      , dialog <#~> identity
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
            , reloadAction args ctx
            ] 
          ]
        ]
      , dialog <#~> identity
      ]