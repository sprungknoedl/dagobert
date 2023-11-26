module Dagobert.Utils.XHR where

import Prelude

import Data.Either (Either(..))
import Data.HTTP.Method (Method(..))
import Effect.Aff (Aff)
import Fetch (fetch)
import Fetch.Yoga.Json (fromJSON)
import Yoga.JSON (class ReadForeign, class WriteForeign, writeJSON)

get :: forall a. ReadForeign a => String -> Aff (Either String a)
get url = do
    { json, text, ok } <- fetch url { method: GET, headers: { "Accept": "application/json" } }
    if ok
        then Right <$> fromJSON json
        else Left <$> text

post :: forall a. WriteForeign a => ReadForeign a => String -> a -> Aff (Either String a)
post url obj = do
    { json, text, ok } <- fetch url { method: POST, headers: { "Accept": "application/json" }, body: writeJSON obj }
    if ok
        then Right <$> fromJSON json
        else Left <$> text

put :: forall a. WriteForeign a => ReadForeign a => String -> a -> Aff (Either String a)
put url obj = do
    { json, text, ok } <- fetch url { method: PUT, headers: { "Accept": "application/json" }, body: writeJSON obj }
    if ok
        then Right <$> fromJSON json
        else Left <$> text

delete :: String -> Aff (Either String Unit)
delete url = do
    { text, ok } <- fetch url { method: DELETE, headers: { "Accept": "application/json" } }
    if ok
        then Right <$> pure unit
        else Left <$> text