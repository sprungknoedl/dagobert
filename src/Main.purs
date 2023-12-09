module Main where

import Prelude

import Control.Alt ((<|>))
import Control.Monad.ST.Class (liftST)
import Dagobert.Route (Route(..), routes)
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
import Data.Tuple.Nested ((/\))
import Deku.Core (fixed)
import Deku.Effect (useHot)
import Deku.Hooks ((<#~>))
import Deku.Toplevel (runInBody)
import Effect (Effect)
import Effect.Aff (Aff, launchAff_)
import Effect.Class (liftEffect)
import FRP.Poll (create)
import Routing.Duplex (parse)
import Routing.Hash (matchesWith)
import Yoga.JSON (class ReadForeign)

main :: Effect Unit
main = do
  _ /\ setRoute /\ route <- liftST $ useHot FourOhFour
  _ /\ setKase /\ kase   <- liftST $ useHot Nothing
  let env = { kase, setKase, route, setRoute }

  cases      <- liftST create

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
  
  runInBody $ fixed
    [ navigationPanel env
    , route <#~> case _ of
      ViewTimeline _        -> eventsPage events env
      ViewAssets _          -> assetsPage assets env
      ViewMalware _         -> malwarePage malware env
      ViewIndicators _      -> indicatorsPage indicators env

      ViewVisualTimeline _  -> loading
      ViewLateralMovement _ -> loading
      ViewActivity _        -> loading

      ViewUsers _           -> usersPage users env
      ViewEvidences _       -> evidencesPage evidences env
      ViewTasks _           -> tasksPage tasks env
      ViewNotes _           -> notesPage notes env

      ViewCases             -> casePage cases env
      FourOhFour            -> loading
    ]

  -- parse hash route & fetch initial data
  _ <- matchesWith
    (map (\e -> e <|> pure FourOhFour) (parse routes))
    (\old new -> when (old /= Just new) $ launchAff_ do
      liftEffect $ setRoute new
      case new of
        ViewTimeline cid        -> fetchData events     ("/api/case/" <> show cid <> "/event")
        ViewAssets cid          -> fetchData assets     ("/api/case/" <> show cid <> "/asset")
        ViewMalware cid         -> fetchData malware    ("/api/case/" <> show cid <> "/malware")
        ViewIndicators cid      -> fetchData indicators ("/api/case/" <> show cid <> "/indicator")

        ViewVisualTimeline _    -> pure unit
        ViewLateralMovement _   -> pure unit
        ViewActivity _          -> pure unit

        ViewUsers cid           -> fetchData users     ("/api/case/" <> show cid <> "/user")
        ViewEvidences cid       -> fetchData evidences ("/api/case/" <> show cid <> "/evidence")
        ViewTasks cid           -> fetchData tasks     ("/api/case/" <> show cid <> "/task")
        ViewNotes cid           -> fetchData notes     ("/api/case/" <> show cid <> "/note")

        ViewCases               -> fetchData cases     ("/api/case")
        FourOhFour              -> pure unit
    )

  pure unit

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