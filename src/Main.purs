module Main where

import Prelude

import Control.Alt ((<|>))
import Control.Monad.ST.Class (liftST)
import Dagobert.Route (Route(..), routeToTitle, routes)
import Dagobert.Utils.HTML (loading)
import Dagobert.Utils.Hooks ((<~))
import Dagobert.Utils.XHR as XHR
import Dagobert.View.AssetsPage (assetsPage)
import Dagobert.View.CasePage (casePage)
import Dagobert.View.EntityPage (PageState(..))
import Dagobert.View.EventsPage (eventsPage)
import Dagobert.View.EvidencesPage (evidencesPage)
import Dagobert.View.IndicatorsPage (indicatorsPage)
import Dagobert.View.MalwarePage (malwarePage)
import Dagobert.View.NavigationPanel (navigationPanel)
import Dagobert.View.NotesPage (notesPage)
import Dagobert.View.TasksPage (tasksPage)
import Dagobert.View.UsersPage (usersPage)
import Data.Either (Either(..))
import Data.Maybe (Maybe(..))
import Deku.Core (fixed)
import Deku.Hooks ((<#~>))
import Deku.Toplevel (runInBody)
import Effect (Effect)
import Effect.Aff (Aff, launchAff_)
import FRP.Poll (create)
import Routing.Duplex (parse)
import Routing.Hash (matchesWith)
import Yoga.JSON (class ReadForeign)

main :: Effect Unit
main = do
  route <- liftST create
  cases <- liftST create
  
  -- Investigation states
  events     <- liftST create
  assets     <- liftST create
  malware    <- liftST create
  indicators <- liftST create

  -- Case Management stats
  users      <- liftST create
  evidences  <- liftST create
  tasks      <- liftST create
  notes      <- liftST create

  let
    fetchData :: forall a r
      . ReadForeign a 
      => { push :: PageState a -> Effect Unit | r } 
      -> String 
      -> Aff Unit
    fetchData state url = do
      state <~ Loading
      resp  <- XHR.get url
      case resp of
        Right list -> state <~ Loaded list
        Left error -> state <~ Error error

  runInBody $ fixed
    [ route.poll <#~> routeToTitle >>> navigationPanel
    , route.poll <#~> case _ of
      ViewTimeline   -> eventsPage events
      ViewAssets     -> assetsPage assets
      ViewMalware    -> malwarePage malware
      ViewIndicators -> indicatorsPage indicators

      ViewVisualTimeline  -> loading
      ViewLateralMovement -> loading
      ViewActivity        -> loading

      ViewUsers      -> usersPage users
      ViewEvidences  -> evidencesPage evidences
      ViewTasks      -> tasksPage tasks
      ViewNotes      -> notesPage notes

      ViewCaseInfo   -> casePage cases
      FourOhFour     -> loading
    ]
  
  -- parse hash route & fetch initial data
  _ <- matchesWith
    (map (\e -> e <|> pure FourOhFour) (parse routes))
    (\old new -> when (old /= Just new) $ launchAff_ do
      route <~ new
      case new of
        ViewTimeline        -> fetchData events "/api/event"
        ViewAssets          -> fetchData assets "/api/asset"
        ViewMalware         -> fetchData malware "/api/malware"
        ViewIndicators      -> fetchData indicators "/api/indicator"

        ViewVisualTimeline  -> pure unit
        ViewLateralMovement -> pure unit
        ViewActivity        -> pure unit

        ViewUsers           -> fetchData users "/api/user"
        ViewEvidences       -> fetchData evidences "/api/evidence"
        ViewTasks           -> fetchData tasks "/api/task"
        ViewNotes           -> fetchData notes "/api/note"

        ViewCaseInfo        -> fetchData cases "/api/case"
        FourOhFour          -> pure unit
      )
  
  pure unit