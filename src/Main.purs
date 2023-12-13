module Main where

import Prelude

import Control.Alt ((<|>))
import Control.Monad.ST.Class (liftST)
import Dagobert.Data.Case (Case)
import Dagobert.Route (Route(..), routes)
import Dagobert.Utils.HTML (loading)
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
          ViewTimeline cid        -> do
            fetchCase setKase cid
            fetchData events     ("/api/cases/" <> show cid <> "/events")
          ViewAssets cid          -> do
            fetchCase setKase cid
            fetchData assets     ("/api/cases/" <> show cid <> "/assets")
          ViewMalware cid         -> do
            fetchCase setKase cid
            fetchData malware    ("/api/cases/" <> show cid <> "/malware")
          ViewIndicators cid      -> do
            fetchCase setKase cid
            fetchData indicators ("/api/cases/" <> show cid <> "/indicators")

          ViewVisualTimeline _    -> pure unit
          ViewLateralMovement _   -> pure unit
          ViewActivity _          -> pure unit

          ViewUsers cid           -> do
              fetchCase setKase cid
              fetchData users     ("/api/cases/" <> show cid <> "/users")
          ViewEvidences cid       -> do
              fetchCase setKase cid
              fetchData evidences ("/api/cases/" <> show cid <> "/evidences")
          ViewTasks cid           -> do
              fetchCase setKase cid
              fetchData tasks     ("/api/cases/" <> show cid <> "/tasks")
          ViewNotes cid           -> do
              fetchCase setKase cid
              fetchData notes     ("/api/cases/" <> show cid <> "/notes")

          ViewCases               -> fetchData cases     ("/api/cases")
          FourOhFour              -> pure unit
    )

  pure unit

fetchCase :: (Maybe Case -> Effect Unit) -> Int -> Aff Unit
fetchCase set cid = do
  eitherCase <- XHR.get $ "/api/cases/" <> show cid
  case eitherCase of
    Right k -> liftEffect $ set (Just k)
    Left _  -> pure unit

fetchData :: forall a r
  . ReadForeign a 
  => { push :: PageState a -> Effect Unit | r } 
  -> String 
  -> Aff Unit
fetchData state url = do
  liftEffect $ state.push Loading
  resp  <- XHR.get url
  case resp of
    Right list -> liftEffect $ state.push (Loaded list)
    Left error -> liftEffect $ state.push (Error error)