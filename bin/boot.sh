export APP_ROOT=`pwd`

#if [ -f $APP_ROOT/gocrowd/htdocs/gocrowd.json ]
#then
#  mv $APP_ROOT/gocrowd/htdocs/gocrowd.json $APP_ROOT/gocrowd/gocrowd.json
#fi

echo "My outgoing IP to Crowd:" `curl ifconfig.co 2> /dev/null`

cd $APP_ROOT/gocrowd
exec $APP_ROOT/gocrowd/gocrowd