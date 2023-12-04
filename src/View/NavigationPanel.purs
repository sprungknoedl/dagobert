module Dagobert.View.NavigationPanel where

import Prelude

import Dagobert.Route (Route(..), routes, routeToTitle)
import Dagobert.Utils.HTML (css)
import Dagobert.Utils.Icons (bug, chatBubble, clipboardCheck, clock, cube, dagobert, desktop, globeEurope, identification, users)
import Deku.Attribute (Attribute)
import Deku.Core (Nut)
import Deku.DOM as D
import Deku.DOM.Attributes as DA
import FRP.Poll (Poll)
import Routing.Duplex (print)

navigationPanel :: String -> Nut
navigationPanel current =
  D.aside [css "h-screen p-4"]
    [ D.div [css "w-64 -p-4" ] [] -- width spacer element
    , D.div [css "w-12 h-12 mt-4 mb-8 mx-auto bg-pink-500 text-slate-800 rounded-lg p-3"] [ dagobert $ css "" ]

    , link ViewCaseInfo   identification

    , D.h3 [ DA.klass_ "mt-4 mb-1 font-bold" ] [ D.text_ "Investigation" ]
    , link ViewTimeline   clock
    , link ViewAssets     desktop
    , link ViewMalware    bug
    , link ViewIndicators globeEurope

    -- , D.h3 [ DA.klass_ "mt-4 mb-1 font-bold" ] [ D.text_ "Reporting" ]
    -- , link ViewVisualTimeline  mempty
    -- , link ViewLateralMovement mempty
    -- , link ViewActivity        mempty

    , D.h3 [ DA.klass_ "mt-4 mb-1 font-bold" ] [ D.text_ "Case Management" ]
    , link ViewUsers     users
    , link ViewEvidences cube
    , link ViewTasks     clipboardCheck
    , link ViewNotes     chatBubble
    ]
  where
  link :: forall r. Route -> (Poll (Attribute (klass :: String | r)) -> Nut) -> Nut
  link dst icon =
    D.a 
      [ css $ if (routeToTitle dst) == current 
          then "block p-2 pl-4 border-l-2 border-pink-500 text-pink-500 font-bold hover:text-white hover:border-slate-100"
          else "block p-2 pl-4 border-l-2 border-slate-600 text-slate-400 hover:text-white hover:border-slate-100"
      , DA.href_ $ "#" <> (print routes dst)
      ] 
      [ icon (css "inline-block mr-2 w-5 h-5")
      , D.text_ (routeToTitle dst)
      ]