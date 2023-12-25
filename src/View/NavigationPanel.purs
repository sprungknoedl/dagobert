module Dagobert.View.NavigationPanel where

import Prelude

import Dagobert.Route (Route(..), routeToTitle, routes)
import Dagobert.Utils.Env (Env)
import Dagobert.Utils.Icons (briefcase, bug, chatBubble, clipboardCheck, clock, cube, dagobert, desktop, fire, globeEurope, identification, users)
import Data.Maybe (Maybe(..), maybe)
import Deku.Attribute (Attribute)
import Deku.Core (Nut, fixed)
import Deku.DOM as D
import Deku.DOM.Attributes as DA
import Deku.Hooks ((<#~>))
import FRP.Poll (Poll)
import Routing.Duplex (print)

navigationPanel :: Env -> Nut
navigationPanel { route, kase } =
  D.aside [ DA.klass_ "h-screen p-4" ]
    [ D.div [ DA.klass_ "w-64 -p-4" ] [] -- width spacer element
    , D.div [ DA.klass_ "w-12 h-12 mt-4 mb-8 mx-auto bg-pink-500 text-slate-800 rounded-lg p-3" ] [ dagobert $ DA.klass_ "" ]

    , D.h3 [ DA.klass_ "mt-4 mb-1 font-bold" ] [ D.text_ "Overview" ]
    , link ViewCases identification
    , D.a
      [ DA.klass_ "block p-2 pl-4 border-l-2 border-slate-600 text-slate-400 hover:text-white hover:border-slate-100" 
      , DA.href_ $ "#" <> (print routes ViewCases)
      ]
      [ fire $ DA.klass_ "inline-block mr-2 w-5 h-5"
      , D.span [] [ D.text_ "Active Case:" ]
      , kase <#~> case _ of
        Just active -> D.span [ DA.klass_ "font-bold text-green-500 block" ] [ D.text_ $ "#" <> (show active.id) <> " - " <> active.name ]
        Nothing     -> D.span [ DA.klass_ "font-bold text-red-500 block" ] [ D.text_ "No case selected" ]
      ]

    , kase <#~> maybe mempty (\c -> fixed
      [ D.h3 [ DA.klass_ "mt-4 mb-1 font-bold" ] [ D.text_ "Investigation" ]
      , link (ViewOverview c.id)   briefcase
      , link (ViewTimeline c.id)   clock
      , link (ViewAssets c.id)     desktop
      , link (ViewMalware c.id)    bug
      , link (ViewIndicators c.id) globeEurope

      -- , D.h3 [ DA.klass_ "mt-4 mb-1 font-bold" ] [ D.text_ "Reporting" ]
      -- , link (ViewVisualTimeline)  mempty
      -- , link (ViewLateralMovement) mempty
      -- , link (ViewActivity)        mempty

      , D.h3 [ DA.klass_ "mt-4 mb-1 font-bold" ] [ D.text_ "Case Management" ]
      , link (ViewUsers c.id)     users
      , link (ViewEvidences c.id) cube
      , link (ViewTasks c.id)     clipboardCheck
      , link (ViewNotes c.id)     chatBubble
      ])
    ]
  where
  link :: forall r. Route -> (Poll (Attribute (klass :: String | r)) -> Nut) -> Nut
  link dst icon =
    D.a 
      [ DA.klass $ route <#> \cur -> if dst == cur
          then "block p-2 pl-4 border-l-2 border-pink-500 text-pink-500 font-bold hover:text-white hover:border-slate-100"
          else "block p-2 pl-4 border-l-2 border-slate-600 text-slate-400 hover:text-white hover:border-slate-100"
      , DA.href_ $ "#" <> (print routes dst)
      ] 
      [ icon (DA.klass_ "inline-block mr-2 w-5 h-5")
      , D.text_ (routeToTitle dst)
      ]