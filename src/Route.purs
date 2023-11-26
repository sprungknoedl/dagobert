module Dagobert.Route where

import Prelude hiding ((/))

import Data.Generic.Rep (class Generic)
import Routing.Duplex (RouteDuplex', root)
import Routing.Duplex.Generic (noArgs, sum)
import Routing.Duplex.Generic.Syntax ((/))

data Route
    = ViewTimeline
    | ViewAssets
    | ViewMalware
    | ViewIndicators

    | ViewVisualTimeline
    | ViewLateralMovement
    | ViewActivity

    | ViewUsers
    | ViewEvidences
    | ViewTasks
    | ViewNotes

    | ViewCaseInfo

    | FourOhFour
    

derive instance genericRoute :: Generic Route _
derive instance eqRoute :: Eq Route
derive instance ordRoute :: Ord Route

routeToTitle :: Route -> String
routeToTitle ViewTimeline   = "Timeline"
routeToTitle ViewAssets     = "Assets"
routeToTitle ViewMalware    = "Malware/Tools"
routeToTitle ViewIndicators = "Indicators"

routeToTitle ViewVisualTimeline  = "Visual Timeline"
routeToTitle ViewLateralMovement = "Lateral Movement"
routeToTitle ViewActivity        = "Activity"

routeToTitle ViewUsers     = "Users"
routeToTitle ViewEvidences = "Evidence"
routeToTitle ViewTasks     = "Tasks"
routeToTitle ViewNotes     = "Notes"

routeToTitle ViewCaseInfo = "Case Information"
routeToTitle FourOhFour   = "404"

routes :: RouteDuplex' Route
routes = root $ sum
  { "ViewTimeline"   : "timeline" / noArgs
  , "ViewAssets"     : "assets" / noArgs
  , "ViewMalware"    : "malware" / noArgs
  , "ViewIndicators" : "indicators" / noArgs

  , "ViewVisualTimeline"  : "visual-timeline" / noArgs
  , "ViewLateralMovement" : "lateral-movement" / noArgs
  , "ViewActivity"        : "activity" / noArgs

  , "ViewUsers"     : "users" / noArgs
  , "ViewEvidences" : "evidence" / noArgs
  , "ViewTasks"     : "tasks" / noArgs
  , "ViewNotes"     : "notes" / noArgs

  , "ViewCaseInfo"  : noArgs
  , "FourOhFour"    : "404" / noArgs
  }