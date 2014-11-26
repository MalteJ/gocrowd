export APP_ROOT=`pwd`

#if [ -f $APP_ROOT/gocrowd/htdocs/gocrowd.json ]
#then
#  mv $APP_ROOT/gocrowd/htdocs/gocrowd.json $APP_ROOT/gocrowd/gocrowd.json
#fi

#touch $APP_ROOT/gocrowd/logs/access.log
#touch $APP_ROOT/gocrowd/logs/error.log

echo "My outgoing IP to Crowd:" `curl ifconfig.co 2> /dev/null`

#(tail -f -n 0 $APP_ROOT/gocrowd/logs/*log &)
#cd $APP_ROOT/gocrowd
exec gocrowd