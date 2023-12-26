module View.OverviewPage where

import Prelude

import Dagobert.Data.Case (Case, CaseStub)
import Dagobert.Utils.HTML (primaryButton, printDate, secondaryButton, secondaryLink)
import Dagobert.Utils.Icons (arrowPath, bolt, briefcase, clipboardCheck, clock, documentArrowDown, inline, pencil, viewfinderCircle)
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

    generateReportAction :: Case -> Nut
    generateReportAction obj =
      secondaryLink [ DA.href_ $ "/api/cases/" <> (show obj.id) <> "/render", DA.target_ "blank" ]
        [ documentArrowDown (DA.klass_ "inline-block mr-1 w-5 h-5")
        , D.text_ "Generate report"
        ]

    topPanel :: String -> Nut
    topPanel title =
      D.main [ DA.klass_ "p-4 grow" ] $
        [ D.nav [ DA.klass_ "flex items-center justify-between mb-4" ]
          [ D.h3 [ DA.klass_ "font-bold text-2xl ml-2" ] [ D.text_ title ]
          , D.div [ DA.klass_ "flex gap-5 items-center" ]
            [ reloadAction ]
          ]
        ]

    pillar :: String -> Array Nut -> Nut
    pillar color body = D.span [ DA.klass_ $ "rounded-full py-2 px-4 mr-2 " <> color ] body

    dlItem :: String -> Array Nut -> Nut
    dlItem title body = fixed
      [ D.dt [ DA.klass_ "float-left clear-left w-32" ] [ D.text_ title ]
      , D.dd [ DA.klass_ "ml-48 mb-6" ] body
      ]

    dlClosed :: Boolean -> Nut
    dlClosed true =  pillar "bg-zinc-600"   [ briefcase inline, D.text_ "Closed" ]
    dlClosed false = pillar "bg-yellow-600" [ briefcase inline, D.text_ "Open" ]

    dlSeverity :: String -> Nut
    dlSeverity t@"Low"    = pillar "bg-yellow-600" [ bolt inline, D.text_ t ]
    dlSeverity t@"Medium" = pillar "bg-amber-600"  [ bolt inline, D.text_ t ]
    dlSeverity t@"High"   = pillar "bg-red-600"    [ bolt inline, D.text_ t ]
    dlSeverity _        = pillar "bg-zinc-600"   [ bolt inline, D.text_ "Unknown" ]

    dlOutcome :: String -> Nut
    dlOutcome t@"False positive"  = pillar "bg-green-600" [ viewfinderCircle inline, D.text_ t ]
    dlOutcome t@"True positive"   = pillar "bg-red-600"   [ viewfinderCircle inline, D.text_ t ]
    dlOutcome t@"Benign positive" = pillar "bg-zinc-600"  [ viewfinderCircle inline, D.text_ t ]
    dlOutcome _                   = pillar "bg-zinc-600"  [ viewfinderCircle inline, D.text_ "Unknown" ]

  state.poll <#~> case _ of
    Loading   -> topPanel $ "Loading ..."
    
    Error err -> topPanel $ "Error: " <> err

    Loaded c  -> D.main [ DA.klass_ "p-4 grow" ]
      [ D.nav [ DA.klass_ "flex items-center justify-between mb-4" ]
        [ D.h3 [ DA.klass_ "font-bold text-2xl ml-2" ] [ D.text_ $ "#" <> (show c.id) <> " - " <> c.name ]
        , D.div [ DA.klass_ "flex gap-5 items-center" ]
          [ generateReportAction c, reloadAction, editAction c ]
        ]
      , D.section [ DA.klass_ "flex gap-6 my-8" ]
        [ D.div [ DA.klass_ "bg-slate-700 p-8 w-1/2" ] 
          [ D.h4 [ DA.klass_ "font-bold text-xl mb-6 pb-2 border-b border-b-slate-600" ] [ D.text_ "Case Overview" ]
          , D.dl_ 
            [ dlItem "Case opened:"    [ pillar "bg-slate-800" [ clock inline, D.text_ $ printDate c.dateAdded ]]
            , dlItem "Case state:"     [ dlClosed c.closed ]
            , dlItem "Classification:" [ pillar "bg-slate-800" [ clipboardCheck inline, D.text_ $ c.classification ]]
            , dlItem "Case severity:"  [ dlSeverity c.severity ]
            , dlItem "Case outcome:"   [ dlOutcome c.outcome ]
            ]
          ]

        , D.div [ DA.klass_ "bg-slate-700 p-8 w-1/2" ]
          [ D.h4 [ DA.klass_ "font-bold text-xl mb-6 pb-2 border-b border-b-slate-600" ] [ D.text_ "Summary" ]
          , D.p_ [ D.text_ c.summary ]
          ]
        ]
      , dialog <#~> identity
      ]