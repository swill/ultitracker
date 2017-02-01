# default primary color
COLOR="#3c8dbc"

TEAM=$1
if [ -z ${TEAM} ]; then
	echo "Using default theme color..."
	echo "Missing team so no config file applied..."
else
	case ${TEAM} in
	    "iris" )
	        COLOR="#313181"
	        ;;
	    "royal" )
			COLOR="#253552"
			;;
	esac

    cp ./config/${TEAM}/config.* .
    cp ./config/${TEAM}/img/* ./static/img/
fi

# rebuild the theme with the appropriate colors
lessc --modify-var="light-blue=${COLOR};" --clean-css build/skin-tracker.less static/theme/admin-lte/dist/css/skins/skin-tracker.min.css
lessc --modify-var="light-blue=${COLOR};" --clean-css build/AdminLTE/build/less/AdminLTE.less static/theme/admin-lte/dist/css/AdminLTE.min.css
