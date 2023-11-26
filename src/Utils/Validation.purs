module Dagobert.Utils.Validation where

import Prelude

import Data.Either (Either(..))
import Data.String (Pattern(..), Replacement(..), null, replace)
import Data.String.Regex (regex, test)
import Data.String.Regex.Flags (noFlags)

type Error = String
type Validator a = a -> Either Error a

date :: String -> Either Error String
date x = do
  let str = """^((?:(\d{4}-\d{2}-\d{2})$|^$"""
  let mod = replace (Pattern " ") (Replacement "T") x
  re <- regex str noFlags
  if test re x
    then Right mod
    else Left "Invalid format, expecting '2006-01-02'."

datetime :: String -> Either Error String
datetime x = do
  let str = """^((?:(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2}:\d{2}(?:\.\d+)?))(Z|[\+-]\d{2}:\d{2}))$|^$"""
  let mod = replace (Pattern " ") (Replacement "T") x
  re <- regex str noFlags
  if test re x
    then Right mod
    else Left "Invalid format, expecting '2006-01-02 15:04:05Z'."

required :: String -> Either Error String
required "" = Left "This field is required."
required x  = Right x

optional :: forall a. a -> Either Error a
optional x = Right x

id :: Int -> Either Error Int
id input
  | input >= 0 = Right input
  | otherwise  = Left "This is an invalid id."

defaultsTo :: String -> String -> Either Error String
defaultsTo d a
  | null a    = Right d
  | otherwise = Right a