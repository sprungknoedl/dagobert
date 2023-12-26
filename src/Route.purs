module Dagobert.Route where

import Prelude hiding ((/))

import Data.Generic.Rep (class Generic)
import Data.Show.Generic (genericShow)
import Routing.Duplex (RouteDuplex', int, segment)
import Routing.Duplex.Generic (noArgs, sum)
import Routing.Duplex.Generic.Syntax ((/))

data Route
    = ViewOverview Int
    | ViewTimeline Int
    | ViewAssets Int
    | ViewMalware Int
    | ViewIndicators Int

    | ViewVisualTimeline Int
    | ViewLateralMovement Int
    | ViewActivity Int

    | ViewUsers Int
    | ViewEvidences Int
    | ViewTasks Int
    | ViewNotes Int

    | ViewCases
    | FourOhFour
    

derive instance genericRoute :: Generic Route _
derive instance eqRoute :: Eq Route
derive instance ordRoute :: Ord Route

instance showRoute :: Show Route where
  show = genericShow

routeToTitle :: Route -> String
routeToTitle (ViewOverview _)        = "Case"
routeToTitle (ViewTimeline _)        = "Timeline"
routeToTitle (ViewAssets _)          = "Assets"
routeToTitle (ViewMalware _)         = "Malware/Tools"
routeToTitle (ViewIndicators _)      = "Indicators"

routeToTitle (ViewVisualTimeline _)  = "Visual Timeline"
routeToTitle (ViewLateralMovement _) = "Lateral Movement"
routeToTitle (ViewActivity _)        = "Activity"

routeToTitle (ViewUsers _)           = "Users"
routeToTitle (ViewEvidences _)       = "Evidence"
routeToTitle (ViewTasks _)           = "Tasks"
routeToTitle (ViewNotes _)           = "Notes"

routeToTitle ViewCases               = "Cases"
routeToTitle FourOhFour              = "404"

routes :: RouteDuplex' Route
routes = sum
  { "ViewOverview"        : "case" / int segment / "overview"
  , "ViewTimeline"        : "case" / int segment / "timeline"
  , "ViewAssets"          : "case" / int segment / "assets"
  , "ViewMalware"         : "case" / int segment / "malware"
  , "ViewIndicators"      : "case" / int segment / "indicators"

  , "ViewVisualTimeline"  : "case" / int segment / "visual-timeline"
  , "ViewLateralMovement" : "case" / int segment / "lateral-movement"
  , "ViewActivity"        : "case" / int segment / "activity"

  , "ViewUsers"           : "case" / int segment / "users"
  , "ViewEvidences"       : "case" / int segment / "evidence"
  , "ViewTasks"           : "case" / int segment / "tasks"
  , "ViewNotes"           : "case" / int segment / "notes"

  , "ViewCases"           : noArgs
  , "FourOhFour"          : "404" / noArgs
  }