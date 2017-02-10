# default primary color
COLOR="#3c8dbc"

TEAM=$1
if [ -z ${TEAM} ]; then
	echo "Using default theme color..."
	echo "Missing team so no config file applied..."
else
	case ${TEAM} in
	    "6ixers" )
	        COLOR="#532D6D"
	        ;;
	    "blaze" )
	        COLOR="#bf0404"
	        ;;
	    "eclipse" )
	        COLOR="#33ACDD"
	        ;;
	    "iris" )
	        COLOR="#313181"
	        ;;
	    "nebula" )
	        COLOR="#33ACDD"
	        ;;
	    "royal" )
			COLOR="#253552"
			;;
	    "stella" )
			COLOR="#139A14"
			;;
	esac

    cp ./config/${TEAM}/config.* .
    cp ./config/${TEAM}/img/* ./static/img/
fi

# rebuild the theme with the appropriate colors
lessc --modify-var="light-blue=${COLOR};" --clean-css build/skin-tracker.less static/theme/admin-lte/dist/css/skins/skin-tracker.min.css
lessc --modify-var="light-blue=${COLOR};" --clean-css build/AdminLTE/build/less/AdminLTE.less static/theme/admin-lte/dist/css/AdminLTE.min.css
