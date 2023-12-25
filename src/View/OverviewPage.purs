module View.OverviewPage where

import Prelude

import Dagobert.Data.Case (Case, CaseStub)
import Dagobert.Utils.HTML (primaryButton, printDate, secondaryButton)
import Dagobert.Utils.Icons (arrowPath, bolt, briefcase, clipboardCheck, clock, pencil, viewfinderCircle)
import Dagobert.Utils.XHR as XHR
import Dagobert.View.CasePage (caseModal)
import Dagobert.View.EntityPage (PageState(..), PollIO)
import Data.Either (Either(..), either)
import Data.Tuple.Nested ((/\))
import Deku.Core (Nut, fixed)
import Deku.DOM as D
import Deku.DOM.Attributes as DA
import Deku.DOM.Listeners as DL
import Deku.Do as Deku
import Deku.Hooks (useState', (<#~>))
import Effect (Effect)
import Effect.Aff (Aff, launchAff_)
import Effect.Class (liftEffect)

overviewPage :: Int -> PollIO (PageState Case) -> Nut
overviewPage cid state = Deku.do
  setDialog /\ dialog <- useState'

  let
    reload :: Aff Unit
    reload = do
      liftEffect $ state.push Loading
      resp <- XHR.get ("/api/cases/" <> show cid)
      case resp of 
        Right elem -> liftEffect $ state.push  (Loaded elem)
        Left err   -> liftEffect $ state.push  (Error err)
      liftEffect $ setDialog mempty

    reloadAction :: Nut
    reloadAction =
      secondaryButton [ DL.runOn_ DL.click $ launchAff_ reload ] 
        [ arrowPath (DA.klass_ "inline-block mr-1 w-5 h-5"), D.text_ "Refresh" ]

    editAction :: Case -> Nut
    editAction obj = Deku.do
      let
        save :: CaseStub -> Effect Unit
        save obj' = do
          state.push $ Loading
          launchAff_   $ XHR.put ("/api/cases/" <> show obj'.id) obj' >>= either
            (\err -> liftEffect $ state.push (Error err))
            (const reload)

        addDialog :: Effect Unit
        addDialog = setDialog $ caseModal { save: save, cancel: setDialog mempty } obj unit

      primaryButton [ DL.runOn_ DL.click $ addDialog ] 
        [ pencil (DA.klass_ "inline-block mr-1 w-5 h-5"), D.text_ "Edit" ]

    topPanel :: String -> Nut
    topPanel title =
      D.main [ DA.klass_ "p-4 grow" ] $
        [ D.nav [ DA.klass_ "flex items-center justify-between mb-4" ]
          [ D.h3 [ DA.klass_ "font-bold text-2xl ml-2" ] [ D.text_ title ]
          , D.div [ DA.klass_ "flex gap-5 items-center" ]
            [ reloadAction ]
          ]
        ]

    dlItem :: String -> Array Nut -> Nut
    dlItem title body = fixed
      [ D.dt [ DA.klass_ "float-left clear-left w-32" ] [ D.text_ title ]
      , D.dd [ DA.klass_ "ml-48 mb-6" ] body
      ]

  state.poll <#~> case _ of
    Loading   -> topPanel $ "Loading ..."
    
    Error err -> topPanel $ "Error: " <> err

    Loaded c  -> D.main [ DA.klass_ "p-4 grow" ]
      [ D.nav [ DA.klass_ "flex items-center justify-between mb-4" ]
        [ D.h3 [ DA.klass_ "font-bold text-2xl ml-2" ] [ D.text_ $ "#" <> (show c.id) <> " - " <> c.name ]
        , D.div [ DA.klass_ "flex gap-5 items-center" ]
          [ reloadAction, editAction c ]
        ]
      , D.section [ DA.klass_ "flex gap-6 my-8" ]
        [ D.div [ DA.klass_ "bg-slate-700 p-8 w-1/2" ] 
          [ D.h4 [ DA.klass_ "font-bold text-xl mb-6 pb-2 border-b border-b-slate-600" ] [ D.text_ "Case Overview" ]
          , D.dl_ 
            [ dlItem "Case opened:" [ D.span [ DA.klass_ "rounded-full bg-slate-800 py-2 px-4 mr-2" ]
              [ clock $ DA.klass_ "inline-block mr-2 w-5 h-5"
              , D.text_ $ printDate c.dateAdded
              ]
            ]
            , dlItem "Case state:" [ D.span [ DA.klass_ "rounded-full bg-yellow-600 py-2 px-4 mr-2" ]
              [ briefcase $ DA.klass_ "inline-block mr-2 w-5 h-5"
              , D.text_ $ "Open"
              ]
            ]
            , dlItem "Classification:" [ D.span [ DA.klass_ "rounded-full bg-slate-800 py-2 px-4 mr-2" ]
              [ clipboardCheck $ DA.klass_ "inline-block mr-2 w-5 h-5"
              , D.text_ $ c.classification
              ]
            ]
            , dlItem "Case severity:" [ D.span [ DA.klass_ "rounded-full bg-red-600 py-2 px-4 mr-2" ]
              [ bolt $ DA.klass_ "inline-block mr-2 w-5 h-5"
              , D.text_ $ "High"
              ]
            ]
            , dlItem "Case outcome:" [ D.span [ DA.klass_ "rounded-full bg-yellow-600 py-2 px-4 mr-2" ]
              [ viewfinderCircle $ DA.klass_ "inline-block mr-2 w-5 h-5"
              , D.text_ $ "In progress"
              ]
            ]
            ]
          ]

        , D.div [ DA.klass_ "bg-slate-700 p-8 w-1/2" ]
          [ D.h4 [ DA.klass_ "font-bold text-xl mb-6 pb-2 border-b border-b-slate-600" ] [ D.text_ "Summary" ]
          , D.p_ [ D.text_ c.summary ]
          ]
        ]
      , dialog <#~> identity
      ]